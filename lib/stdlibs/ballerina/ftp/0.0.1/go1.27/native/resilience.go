// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package native

import (
	"errors"
	"io"
	"math"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/values"
	"github.com/pkg/sftp"
)

func sleep(platform pal.Platform, d time.Duration) {
	done := make(chan struct{})
	platform.Time.AfterFunc(d, func() { close(done) })
	<-done
}

// withRetry runs op once and then up to count more times, waiting interval seconds before the
// first retry and multiplying the wait by backOffFactor (capped at maxWaitInterval) after that.
// Only transient failures are retried (ballerina-platform/ballerina-library#9385).
func withRetry[T any](platform pal.Platform, settings *retrySettings, name string, op func() (T, error)) (T, error) {
	result, err := op()
	if err == nil || settings == nil || !isRetryable(err) {
		return result, err
	}
	wait := settings.interval
	for attempt := int64(1); attempt <= settings.count; attempt++ {
		if attempt > 1 {
			wait = math.Min(wait*settings.backOffFactor, settings.maxWaitInterval)
		}
		sleep(platform, time.Duration(int64(wait*1000))*time.Millisecond)
		result, err = op()
		if err == nil || !isRetryable(err) {
			return result, err
		}
	}
	var zero T
	return zero, &ftpError{
		kind:    kindAllRetryAttemptsFailed,
		message: "Operation '" + name + "' failed after " + strconv.FormatInt(settings.count, 10) + " retry attempts: " + asFtpError(err).message,
		cause:   err,
	}
}

// isRetryable reports whether a failure may succeed on a later attempt: a 4xx reply from the
// server or a broken or unreachable connection.
func isRetryable(err error) bool {
	switch asFtpError(err).kind {
	case kindServiceUnavailable:
		return true
	case kindFileNotFound, kindFileAlreadyExists, kindInvalidConfig, kindContentBinding,
		kindCircuitBreakerOpen, kindAllRetryAttemptsFailed:
		return false
	}
	var reply *textproto.Error
	if errors.As(err, &reply) {
		return isServiceUnavailableCode(reply.Code)
	}
	var netErr net.Error
	if errors.As(err, &netErr) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) || errors.Is(err, sftp.ErrSSHFxConnectionLost) ||
		errors.Is(err, sftp.ErrSSHFxNoConnection) {
		return true
	}
	category, ok := categorize(err)
	return ok && category != categoryAuthentication
}

type circuitState int

const (
	circuitClosed circuitState = iota
	circuitOpen
	circuitHalfOpen
)

type failureCategory string

const (
	categoryConnection     failureCategory = "CONNECTION_ERROR"
	categoryAuthentication failureCategory = "AUTHENTICATION_ERROR"
	categoryTransient      failureCategory = "TRANSIENT_ERROR"
	categoryAll            failureCategory = "ALL_ERRORS"
)

type bucket struct {
	total    int64
	failures int64
}

// circuitBreaker ports the jBallerina CircuitBreaker and CircuitHealth: outcomes are counted in
// time buckets over a rolling window, and the circuit opens once the volume threshold is met
// and the failure ratio exceeds the threshold.
type circuitBreaker struct {
	mu         sync.Mutex
	now        func() time.Time
	volume     int64
	threshold  float32
	resetMs    int64
	windowMs   int64
	bucketMs   int64
	categories map[failureCategory]bool

	state           circuitState
	trialInProgress bool
	buckets         []bucket
	startMs         int64
	lastRequestMs   int64
	lastUsedBucket  int
	lastErrorMs     int64
	hasError        bool
}

func millis(f float64) int64 {
	return int64(f * 1000)
}

func newCircuitBreaker(config *values.Map, now func() time.Time) (*circuitBreaker, error) {
	window := getMap(config, "rollingWindow")
	cb := &circuitBreaker{
		now:        now,
		volume:     getInt(window, "requestVolumeThreshold"),
		threshold:  float32(getFloat(config, "failureThreshold")),
		resetMs:    millis(getFloat(config, "resetTime")),
		windowMs:   millis(getFloat(window, "timeWindow")),
		bucketMs:   millis(getFloat(window, "bucketSize")),
		categories: map[failureCategory]bool{},
	}
	for _, c := range getStrings(config, "failureCategories") {
		cb.categories[failureCategory(c)] = true
	}
	switch {
	case len(cb.categories) == 0:
		return nil, newError(kindInvalidConfig, "Circuit breaker failureCategories must not be empty")
	case cb.threshold < 0 || cb.threshold > 1:
		return nil, newError(kindInvalidConfig, "Circuit breaker failureThreshold must be between 0.0 and 1.0")
	case cb.volume <= 0:
		return nil, newError(kindInvalidConfig, "Circuit breaker requestVolumeThreshold must be greater than 0")
	case cb.resetMs <= 0:
		return nil, newError(kindInvalidConfig, "Circuit breaker resetTime must be greater than 0")
	case cb.bucketMs <= 0:
		return nil, newError(kindInvalidConfig, "Circuit breaker bucketSize must be greater than 0")
	case cb.windowMs < cb.bucketMs:
		return nil, newError(kindInvalidConfig, "Circuit breaker timeWindow must be greater than or equal to bucketSize")
	case cb.windowMs%cb.bucketMs != 0:
		return nil, newError(kindInvalidConfig, "Circuit breaker timeWindow must be evenly divisible by bucketSize")
	}
	cb.buckets = make([]bucket, cb.windowMs/cb.bucketMs)
	cb.startMs = cb.nowMs()
	cb.lastRequestMs = cb.startMs
	return cb, nil
}

func (cb *circuitBreaker) nowMs() int64 {
	return cb.now().UnixMilli()
}

func (cb *circuitBreaker) currentBucket() int {
	elapsed := (cb.nowMs() - cb.startMs) % cb.windowMs
	return int((elapsed / cb.bucketMs) % int64(len(cb.buckets)))
}

func (cb *circuitBreaker) resetAll() {
	for i := range cb.buckets {
		cb.buckets[i] = bucket{}
	}
}

func (cb *circuitBreaker) prepareRollingWindow() {
	if cb.nowMs()-cb.lastRequestMs > cb.windowMs {
		cb.resetAll()
		return
	}
	current := cb.currentBucket()
	switch {
	case current < cb.lastUsedBucket:
		for i := 0; i <= current; i++ {
			cb.buckets[i] = bucket{}
		}
		for i := cb.lastUsedBucket + 1; i < len(cb.buckets); i++ {
			cb.buckets[i] = bucket{}
		}
	case current > cb.lastUsedBucket:
		for i := cb.lastUsedBucket + 1; i <= current; i++ {
			cb.buckets[i] = bucket{}
		}
	}
}

func (cb *circuitBreaker) totals() (int64, int64) {
	var total, failures int64
	for _, b := range cb.buckets {
		total += b.total
		failures += b.failures
	}
	return total, failures
}

func (cb *circuitBreaker) updateState() {
	cb.prepareRollingWindow()
	switch cb.state {
	case circuitOpen:
		if !cb.hasError || cb.nowMs()-cb.lastErrorMs >= cb.resetMs {
			cb.state = circuitHalfOpen
			cb.trialInProgress = false
		}
	case circuitClosed:
		total, failures := cb.totals()
		if total >= cb.volume {
			ratio := float32(0)
			if total > 0 {
				ratio = float32(failures) / float32(total)
			}
			if ratio > cb.threshold {
				cb.state = circuitOpen
			}
		}
	}
}

func (cb *circuitBreaker) isOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case circuitClosed:
		return false
	case circuitHalfOpen:
		return cb.trialInProgress
	}
	cb.updateState()
	if cb.state == circuitHalfOpen {
		return cb.trialInProgress
	}
	return cb.state == circuitOpen
}

func (cb *circuitBreaker) recordRequestStart() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.updateState()
	if cb.state == circuitOpen || (cb.state == circuitHalfOpen && cb.trialInProgress) {
		return
	}
	if cb.state == circuitHalfOpen {
		cb.trialInProgress = true
	}
	cb.prepareRollingWindow()
	current := cb.currentBucket()
	cb.buckets[current].total++
	cb.lastUsedBucket = current
	cb.lastRequestMs = cb.nowMs()
}

func (cb *circuitBreaker) recordFailure() {
	cb.buckets[cb.currentBucket()].failures++
	cb.lastErrorMs = cb.nowMs()
	cb.hasError = true
}

func (cb *circuitBreaker) recordOutcome(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == circuitHalfOpen {
		if err != nil && cb.countsAsFailure(err) {
			cb.recordFailure()
			cb.state = circuitOpen
		} else {
			cb.state = circuitClosed
			cb.resetAll()
		}
		cb.trialInProgress = false
		return
	}
	if err != nil && cb.countsAsFailure(err) {
		cb.recordFailure()
	}
	cb.updateState()
}

func (cb *circuitBreaker) openError() error {
	cb.mu.Lock()
	remaining := int64(0)
	if cb.hasError {
		remaining = max(0, cb.resetMs-(cb.nowMs()-cb.lastErrorMs))
	}
	cb.mu.Unlock()
	secondsLeft := math.Floor(float64(remaining)/1000.0 + 0.5)
	return newError(kindCircuitBreakerOpen, "FTP server unavailable. Circuit breaker is OPEN. Retry in "+
		strconv.FormatFloat(secondsLeft, 'f', 0, 64)+" seconds.")
}

func (cb *circuitBreaker) countsAsFailure(err error) bool {
	if cb.categories[categoryAll] {
		return true
	}
	category, ok := categorize(err)
	return ok && cb.categories[category]
}

// categorize mirrors FailureCategorizer as it sees client errors: the cause chain of the
// Ballerina error (or its own message when it has no cause) is matched by reply code and text.
func categorize(err error) (failureCategory, bool) {
	typed := asFtpError(err)
	messages := []string{typed.message}
	if typed.cause != nil {
		messages = nil
		for c := typed.cause; c != nil; c = errors.Unwrap(c) {
			messages = append(messages, c.Error())
		}
	}
	for _, message := range messages {
		if category, ok := categorizeMessage(message); ok {
			return category, true
		}
	}
	return "", false
}

var (
	connectionPhrases = []string{"connection refused", "connection timed out", "connect timed out",
		"network is unreachable", "no route to host", "host is unreachable", "unknown host", "connect failed",
		"failed to connect", "could not connect"}
	authenticationPhrases = []string{"authentication failed", "invalid credentials", "auth fail", "login failed",
		"access denied", "not authorized", "login incorrect"}
	transientPhrases = []string{"connection reset", "broken pipe", "connection closed", "unexpected end",
		"stream closed", "socket closed", "remote host closed"}
)

func categorizeMessage(message string) (failureCategory, bool) {
	if code, ok := extractFtpCode(message); ok {
		if isServiceUnavailableCode(code) {
			return categoryTransient, true
		}
		if code == 530 {
			return categoryAuthentication, true
		}
	}
	lower := strings.ToLower(message)
	for _, group := range []struct {
		phrases  []string
		category failureCategory
	}{
		{connectionPhrases, categoryConnection},
		{authenticationPhrases, categoryAuthentication},
		{transientPhrases, categoryTransient},
	} {
		for _, phrase := range group.phrases {
			if strings.Contains(lower, phrase) {
				return group.category, true
			}
		}
	}
	return "", false
}

// guard runs a client operation through the circuit breaker, when one is configured.
func guard[T any](cb *circuitBreaker, op func() (T, error)) (T, error) {
	if cb == nil {
		return op()
	}
	if cb.isOpen() {
		var zero T
		return zero, cb.openError()
	}
	cb.recordRequestStart()
	result, err := op()
	cb.recordOutcome(err)
	return result, err
}
