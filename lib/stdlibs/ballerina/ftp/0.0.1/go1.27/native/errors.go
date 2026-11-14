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
	"net/textproto"
	"regexp"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type errorKind string

const (
	kindError                  errorKind = "Error"
	kindConnection             errorKind = "ConnectionError"
	kindFileNotFound           errorKind = "FileNotFoundError"
	kindFileAlreadyExists      errorKind = "FileAlreadyExistsError"
	kindInvalidConfig          errorKind = "InvalidConfigError"
	kindServiceUnavailable     errorKind = "ServiceUnavailableError"
	kindContentBinding         errorKind = "ContentBindingError"
	kindAllRetryAttemptsFailed errorKind = "AllRetryAttemptsFailedError"
	kindCircuitBreakerOpen     errorKind = "CircuitBreakerOpenError"
)

// ftpError is the Go-side form of an ftp:Error; toBal turns it into the distinct Ballerina
// error type named by kind.
type ftpError struct {
	kind     errorKind
	message  string
	cause    error
	filePath *string
	content  []byte
}

func (e *ftpError) Error() string { return e.message }

func (e *ftpError) Unwrap() error { return e.cause }

func newError(kind errorKind, message string) *ftpError {
	return &ftpError{kind: kind, message: message}
}

// balCause carries a Ballerina error value through Go code so it can become the cause of
// an ftp:Error unchanged.
type balCause struct {
	value *values.Error
}

func (c *balCause) Error() string { return c.value.Message }

var ftpCodePattern = regexp.MustCompile(`\b([1-5]\d{2})\b`)

// ftpReplyCode prefers the code of a server reply in the cause chain. Otherwise it takes the last
// 3-digit word of the message, as a reply is appended after any address such as 127.0.0.1; jBallerina
// takes the first (ballerina-library#9383).
func ftpReplyCode(message string, cause error) (int, bool) {
	var reply *textproto.Error
	if errors.As(cause, &reply) {
		return reply.Code, true
	}
	matches := ftpCodePattern.FindAllStringSubmatch(message, -1)
	if len(matches) == 0 {
		return 0, false
	}
	code, _ := strconv.Atoi(matches[len(matches)-1][1])
	return code, true
}

func isServiceUnavailableCode(code int) bool {
	switch code {
	case 421, 425, 426, 450, 451, 452:
		return true
	}
	return false
}

// fullMessage joins the messages of an error chain the way FtpErrorCodeAnalyzer does.
func fullMessage(message string, cause error) string {
	parts := []string{message}
	for c := cause; c != nil; c = errors.Unwrap(c) {
		parts = append(parts, c.Error())
		if _, ok := c.(*ftpError); ok {
			break
		}
	}
	return strings.Join(parts, " Caused by: ")
}

// classify maps an untyped failure to an error kind by its reply code.
func classify(message string, cause error) errorKind {
	code, ok := ftpReplyCode(fullMessage(message, cause), cause)
	if !ok {
		return kindError
	}
	switch {
	case isServiceUnavailableCode(code):
		return kindServiceUnavailable
	case code == 530:
		return kindConnection
	case code == 550:
		return kindFileNotFound
	}
	return kindError
}

// operationError wraps a transport failure. A failure that already is an ftpError keeps its
// kind; anything else is classified by reply code.
func operationError(message string, cause error) *ftpError {
	var typed *ftpError
	if errors.As(cause, &typed) && typed.kind != kindError {
		return &ftpError{kind: typed.kind, message: message, cause: cause}
	}
	return &ftpError{kind: classify(message, cause), message: message, cause: cause}
}

func asFtpError(err error) *ftpError {
	var typed *ftpError
	if errors.As(err, &typed) {
		return typed
	}
	return &ftpError{kind: classify(err.Error(), nil), message: err.Error()}
}

// toBal builds the Ballerina error through the module's createError helper, since the distinct
// error types are only constructible from Ballerina code.
func (m *ftpModule) toBal(ctx *extern.Context, err error) values.BalValue {
	e := asFtpError(err)
	var cause values.BalValue
	switch c := e.cause.(type) {
	case nil:
	case *balCause:
		cause = c.value
	case *ftpError:
		cause = m.toBal(ctx, c)
	default:
		cause = values.NewErrorWithMessage(maskURLPassword(c.Error()))
	}
	var filePath values.BalValue
	if e.filePath != nil {
		filePath = *e.filePath
	}
	var content values.BalValue
	if e.content != nil {
		content = m.types.readonlyBytes(e.content)
	}
	handle, ok := ctx.LookupFunction(orgName, moduleName, "createError")
	if !ok {
		return values.NewErrorWithMessage(e.message)
	}
	result, invokeErr := ctx.InvokeFunction(handle, []values.BalValue{string(e.kind), maskURLPassword(e.message), cause, filePath, content})
	if invokeErr != nil {
		return values.NewErrorWithMessage(e.message)
	}
	return result
}

func (m *ftpModule) errorResult(ctx *extern.Context, err error) (values.BalValue, error) {
	if err == nil {
		return nil, nil
	}
	return m.toBal(ctx, err), nil
}
