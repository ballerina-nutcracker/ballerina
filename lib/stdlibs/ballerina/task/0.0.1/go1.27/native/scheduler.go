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
	"math"
	"sort"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

const (
	repeatIndefinitely = -1
	noFireTime         = math.MinInt64

	triggerGroup    = "trigger"
	logTriggerGroup = "log"
)

var errSchedulerShutdown = errors.New("The Scheduler has been shutdown.")

type triggerState int

const (
	stateNone triggerState = iota
	stateWaiting
	statePaused
	stateComplete
)

// triggerSpec is the schedule a job was created with. Like the Quartz trigger
// kept by jBallerina's TaskManager, it stays pristine so a reconfigured worker
// pool reschedules every live job from its original schedule.
type triggerSpec struct {
	name          string
	startMs       int64
	endMs         int64
	intervalMs    int64
	repeatCount   int64
	group         string
	ignoreMisfire bool
	job           *jobDetail
}

// trigger is the mutable firing state of a triggerSpec stored in one scheduler,
// mirroring Quartz's SimpleTriggerImpl.
type trigger struct {
	spec           *triggerSpec
	startMs        int64
	repeatCount    int64
	timesTriggered int64
	nextMs         int64
	paused         bool
	complete       bool
}

func newTrigger(spec *triggerSpec) *trigger {
	return &trigger{
		spec:        spec,
		startMs:     spec.startMs,
		repeatCount: spec.repeatCount,
		nextMs:      spec.startMs,
	}
}

func (t *trigger) hasEnd() bool {
	return t.spec.endMs != noFireTime
}

func (t *trigger) fireTimeAfter(afterMs int64) int64 {
	if t.repeatCount != repeatIndefinitely && t.timesTriggered > t.repeatCount {
		return noFireTime
	}
	if t.repeatCount == 0 && afterMs >= t.startMs {
		return noFireTime
	}
	endMs := int64(math.MaxInt64)
	if t.hasEnd() {
		endMs = t.spec.endMs
	}
	if endMs <= afterMs {
		return noFireTime
	}
	if afterMs < t.startMs {
		return t.startMs
	}
	executed := (afterMs-t.startMs)/t.spec.intervalMs + 1
	if t.repeatCount != repeatIndefinitely && executed > t.repeatCount {
		return noFireTime
	}
	fireMs := t.startMs + executed*t.spec.intervalMs
	if endMs <= fireMs {
		return noFireTime
	}
	return fireMs
}

func (t *trigger) triggered() {
	t.timesTriggered++
	t.nextMs = t.fireTimeAfter(t.nextMs)
}

// updateAfterMisfire applies Quartz's SimpleTrigger smart misfire policy.
func (t *trigger) updateAfterMisfire(nowMs int64) {
	switch t.repeatCount {
	case 0:
		t.nextMs = nowMs
	case repeatIndefinitely:
		newMs := t.fireTimeAfter(nowMs)
		if newMs != noFireTime {
			t.timesTriggered += (newMs - t.nextMs) / t.spec.intervalMs
		}
		t.nextMs = newMs
	default:
		t.repeatCount -= t.timesTriggered
		t.timesTriggered = 0
		if t.hasEnd() && t.spec.endMs < nowMs {
			t.nextMs = noFireTime
			return
		}
		t.startMs = nowMs
		t.nextMs = nowMs
	}
}

// scheduler mirrors one Quartz scheduler instance backed by a RAMJobStore.
// Every method must be called with the owning manager's mutex held.
type scheduler struct {
	mgr          *taskManager
	thresholdMs  int64
	started      bool
	shutdown     bool
	triggers     map[string]*trigger
	pausedGroups map[string]bool
	timer        pal.Timer
}

func newScheduler(mgr *taskManager, thresholdMs int64) *scheduler {
	return &scheduler{
		mgr:          mgr,
		thresholdMs:  thresholdMs,
		triggers:     map[string]*trigger{},
		pausedGroups: map[string]bool{},
	}
}

func (s *scheduler) validateState() error {
	if s.shutdown {
		return errSchedulerShutdown
	}
	return nil
}

func (s *scheduler) scheduleJob(spec *triggerSpec) error {
	if err := s.validateState(); err != nil {
		return err
	}
	if spec.repeatCount != 0 && spec.intervalMs < 1 {
		return errors.New("Repeat Interval cannot be zero.")
	}
	if _, exists := s.triggers[spec.name]; exists {
		return errors.New("Unable to store Job : 'DEFAULT." + spec.name +
			"', because one already exists with this identification.")
	}
	t := newTrigger(spec)
	t.paused = s.pausedGroups[spec.group]
	s.triggers[spec.name] = t
	s.started = true
	s.arm()
	return nil
}

func (s *scheduler) unscheduleJob(name string) error {
	if err := s.validateState(); err != nil {
		return err
	}
	delete(s.triggers, name)
	s.arm()
	return nil
}

func (s *scheduler) triggerState(name string) (triggerState, error) {
	if err := s.validateState(); err != nil {
		return stateNone, err
	}
	t, ok := s.triggers[name]
	switch {
	case !ok:
		return stateNone, nil
	case t.complete:
		return stateComplete, nil
	case t.paused:
		return statePaused, nil
	default:
		return stateWaiting, nil
	}
}

func (s *scheduler) pauseJob(name string) error {
	if err := s.validateState(); err != nil {
		return err
	}
	if t, ok := s.triggers[name]; ok {
		pauseTrigger(t)
	}
	s.arm()
	return nil
}

func (s *scheduler) resumeJob(name string) error {
	if err := s.validateState(); err != nil {
		return err
	}
	if t, ok := s.triggers[name]; ok {
		s.resumeTrigger(t)
	}
	s.arm()
	return nil
}

func (s *scheduler) pauseAll() error {
	if err := s.validateState(); err != nil {
		return err
	}
	for _, t := range s.triggers {
		s.pausedGroups[t.spec.group] = true
		pauseTrigger(t)
	}
	s.arm()
	return nil
}

func (s *scheduler) resumeAll() error {
	if err := s.validateState(); err != nil {
		return err
	}
	for _, t := range s.sortedTriggers() {
		s.resumeTrigger(t)
	}
	s.pausedGroups = map[string]bool{}
	s.arm()
	return nil
}

func pauseTrigger(t *trigger) {
	if !t.complete {
		t.paused = true
	}
}

func (s *scheduler) resumeTrigger(t *trigger) {
	if !t.paused {
		return
	}
	t.paused = false
	s.applyMisfire(t, s.mgr.nowMs())
}

func (s *scheduler) shutdownNow() {
	s.shutdown = true
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}

// applyMisfire reports whether t had missed its fire time by more than the
// misfire threshold and was rescheduled according to its misfire policy.
func (s *scheduler) applyMisfire(t *trigger, nowMs int64) bool {
	if t.nextMs == noFireTime || t.nextMs > nowMs-s.thresholdMs || t.spec.ignoreMisfire {
		return false
	}
	if t.spec.group == logTriggerGroup {
		s.mgr.logMisfire(t)
	}
	prevMs := t.nextMs
	t.updateAfterMisfire(nowMs)
	if t.nextMs == noFireTime {
		t.complete = true
		return true
	}
	return t.nextMs != prevMs
}

func (s *scheduler) sortedTriggers() []*trigger {
	list := make([]*trigger, 0, len(s.triggers))
	for _, t := range s.triggers {
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].nextMs != list[j].nextMs {
			return list[i].nextMs < list[j].nextMs
		}
		return list[i].spec.name < list[j].spec.name
	})
	return list
}

func (s *scheduler) nextDue(nowMs int64) *trigger {
	for _, t := range s.sortedTriggers() {
		if t.paused || t.complete || t.nextMs == noFireTime {
			continue
		}
		if t.nextMs <= nowMs {
			return t
		}
		return nil
	}
	return nil
}

func (s *scheduler) arm() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if !s.started || s.shutdown {
		return
	}
	earliest := int64(noFireTime)
	for _, t := range s.triggers {
		if t.paused || t.complete || t.nextMs == noFireTime {
			continue
		}
		if earliest == noFireTime || t.nextMs < earliest {
			earliest = t.nextMs
		}
	}
	if earliest == noFireTime {
		return
	}
	delay := max(earliest-s.mgr.nowMs(), 0)
	s.timer = s.mgr.pal.Time.AfterFunc(time.Duration(delay)*time.Millisecond, s.fireDue)
}

// fireDue fires every trigger whose fire time has arrived. Like jBallerina's
// TaskJob, each execution is handed off to its own goroutine, so the worker
// pool never delays a fire.
func (s *scheduler) fireDue() {
	s.mgr.mu.Lock()
	if s.shutdown {
		s.mgr.mu.Unlock()
		return
	}
	var fired []*triggerSpec
	nowMs := s.mgr.nowMs()
	for t := s.nextDue(nowMs); t != nil; t = s.nextDue(nowMs) {
		if s.applyMisfire(t, nowMs) {
			continue
		}
		t.triggered()
		if t.nextMs == noFireTime {
			delete(s.triggers, t.spec.name)
		}
		fired = append(fired, t.spec)
	}
	s.arm()
	s.mgr.mu.Unlock()
	for _, spec := range fired {
		go s.mgr.execute(s, spec)
	}
}
