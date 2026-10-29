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

// These tests cover output a corpus -v test cannot assert: the error and misfire
// logs carry wall-clock timestamps.
package native

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type capture struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (c *capture) write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Write(p)
}

func (c *capture) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

func newTestManager() (*taskManager, *capture, *capture) {
	stdout, stderr := &capture{}, &capture{}
	var platform pal.Platform
	platform.IO.Stdout = stdout.write
	platform.IO.Stderr = stderr.write
	platform.Time.Now = time.Now
	platform.Time.AfterFunc = func(d time.Duration, f func()) pal.Timer { return time.AfterFunc(d, f) }
	return newTaskManager(nil, platform), stdout, stderr
}

func serviceSpec(name, policy string, retry *retryConfig) *triggerSpec {
	return &triggerSpec{
		name:        name,
		startMs:     time.Now().Add(time.Hour).UnixMilli(),
		endMs:       noFireTime,
		repeatCount: repeatIndefinitely,
		intervalMs:  1000,
		group:       triggerGroup,
		job: &jobDetail{
			obj:         values.NewObject(semtypes.Object, nil, nil, nil, nil),
			errorPolicy: policy,
			retry:       retry,
			intervalSec: 1,
			intervalInt: 1,
			isService:   true,
		},
	}
}

var logLine = regexp.MustCompile(`^time = \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3} message = `)

func TestNotifyFailureLogsAndTerminates(t *testing.T) {
	t.Parallel()
	m, stdout, _ := newTestManager()
	s := newScheduler(m, defaultThresholdMs)
	spec := serviceSpec("job-1", policyLogAndTerminate, nil)
	s.triggers[spec.name] = newTrigger(spec)

	m.notifyFailure(s, spec, "boom")

	out := stdout.String()
	if !logLine.MatchString(out) || !strings.HasSuffix(out, "Unable to execute the job[job-1]. boom\n") {
		t.Fatalf("unexpected log line %q", out)
	}
	if _, ok := s.triggers[spec.name]; ok {
		t.Fatal("LOG_AND_TERMINATE should unschedule the trigger")
	}

	s.shutdownNow()
	m.notifyFailure(s, spec, "again")
	if !strings.Contains(stdout.String(), "org.quartz.SchedulerException: The Scheduler has been shutdown.") {
		t.Fatalf("expected the unschedule failure to be logged, got %q", stdout.String())
	}
}

func TestLogMisfire(t *testing.T) {
	t.Parallel()
	m, _, stderr := newTestManager()
	spec := serviceSpec("7", policyLogAndContinue, nil)
	m.jobs[7] = spec
	m.logMisfire(newTrigger(spec))
	m.logMisfire(newTrigger(serviceSpec("svc", policyLogAndContinue, nil)))

	lines := strings.Split(strings.TrimSuffix(stderr.String(), "\n"), "\n")
	if len(lines) != 2 || !logLine.MatchString(lines[0]) {
		t.Fatalf("unexpected misfire log %q", stderr.String())
	}
	if !strings.HasSuffix(lines[0], "couldn't get the resources to execute the job[7]") ||
		!strings.HasSuffix(lines[1], "couldn't get the resources to execute the job[null]") {
		t.Fatalf("unexpected job ids in misfire log %q", stderr.String())
	}
}

func TestLogAndIgnoreMisfireIsLogged(t *testing.T) {
	t.Parallel()
	m, _, stderr := newTestManager()
	s := newScheduler(m, 1)
	spec := serviceSpec("svc", policyLogAndContinue, nil)
	spec.group = logTriggerGroup
	spec.startMs = m.nowMs() - 5000
	tr := newTrigger(spec)
	if !s.applyMisfire(tr, m.nowMs()) {
		t.Fatal("a trigger 5s late with a 1ms threshold should misfire")
	}
	if !strings.Contains(stderr.String(), "has ignored as couldn't get the resources") {
		t.Fatalf("expected a misfire log, got %q", stderr.String())
	}
}

func TestGetRunningJobsReportsShutdownScheduler(t *testing.T) {
	t.Parallel()
	m, _, stderr := newTestManager()
	m.sched = newScheduler(m, defaultThresholdMs)
	m.jobs[1] = serviceSpec("1", policyLogAndContinue, nil)
	m.sched.shutdownNow()

	if ids := m.getRunningJobs(); len(ids) != 0 {
		t.Fatalf("expected no ids, got %v", ids)
	}
	if !strings.HasSuffix(stderr.String(), "message = The Scheduler has been shutdown.\n") {
		t.Fatalf("expected the shutdown error on stderr, got %q", stderr.String())
	}
}

func TestPanicMessage(t *testing.T) {
	t.Parallel()
	if panicMessage(values.NewErrorWithMessage("e")) != "e" || panicMessage(errors.New("g")) != "g" || panicMessage(3) != "3" {
		t.Fatal("unexpected panic message")
	}
}
