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
	"math/rand/v2"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	defaultThreadCount = 5
	defaultThresholdMs = 5000
	jobIDBound         = 1000000

	policyLogAndTerminate = "LOG_AND_TERMINATE"
	policyLogAndContinue  = "LOG_AND_CONTINUE"
	policyTerminate       = "TERMINATE"
	policyWait            = "WAIT"
	policyLogAndIgnore    = "LOG_AND_IGNORE"
)

type schedulerProps struct {
	threadCount int64
	thresholdMs int64
}

// jobDetail is what a trigger executes: a task:Job (or task:Service) object
// plus the policies and retry settings that govern a failed execution.
type jobDetail struct {
	obj         *values.Object
	errorPolicy string
	retry       *retryConfig
	intervalSec float64
	intervalInt int64
	isService   bool
}

// taskManager is the per-runtime counterpart of jBallerina's TaskManager
// singleton: it owns the current scheduler and the jobs scheduled through it.
type taskManager struct {
	rt       *runtime.Runtime
	pal      pal.Platform
	mu       sync.Mutex
	sched    *scheduler
	pending  *schedulerProps
	jobs     map[int64]*triggerSpec
	services map[string]*triggerSpec
}

func newTaskManager(rt *runtime.Runtime, platform pal.Platform) *taskManager {
	return &taskManager{
		rt:       rt,
		pal:      platform,
		jobs:     map[int64]*triggerSpec{},
		services: map[string]*triggerSpec{},
	}
}

func (m *taskManager) nowMs() int64 {
	return m.pal.Time.Now().UnixMilli()
}

func (m *taskManager) createScheduler(props schedulerProps) (*scheduler, error) {
	if props.thresholdMs < 1 {
		return nil, balError("Cannot create the Scheduler.JobStore class 'org.quartz.simpl.RAMJobStore' props could not be configured.")
	}
	if props.threadCount <= 0 {
		return nil, balError("Cannot create the Scheduler.Thread count must be > 0")
	}
	return newScheduler(m, props.thresholdMs), nil
}

func (m *taskManager) getScheduler() (*scheduler, error) {
	if m.pending != nil {
		s, err := m.createScheduler(*m.pending)
		if err != nil {
			return nil, err
		}
		m.sched = s
		m.pending = nil
		return s, nil
	}
	if m.sched == nil || m.sched.shutdown {
		s, err := m.createScheduler(schedulerProps{threadCount: defaultThreadCount, thresholdMs: defaultThresholdMs})
		if err != nil {
			return nil, err
		}
		m.sched = s
	}
	return m.sched, nil
}

func (m *taskManager) configure(props schedulerProps) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.runningJobs(); err != nil {
		return err
	}
	if _, err := m.runningServices(); err != nil {
		return err
	}
	if m.sched != nil {
		m.sched.shutdownNow()
	}
	if len(m.jobs) == 0 && len(m.services) == 0 {
		m.pending = &props
		return nil
	}
	s, err := m.createScheduler(props)
	if err != nil {
		return err
	}
	m.sched = s
	for _, id := range sortedJobIDs(m.jobs) {
		if err := s.scheduleJob(m.jobs[id]); err != nil {
			return err
		}
	}
	for _, name := range sortedNames(m.services) {
		if err := s.scheduleJob(m.services[name]); err != nil {
			return err
		}
	}
	return nil
}

// runningJobs drops the jobs whose trigger has completed or been removed and
// returns the remaining job ids in ascending order.
func (m *taskManager) runningJobs() ([]int64, error) {
	for _, id := range sortedJobIDs(m.jobs) {
		state, err := m.sched.triggerState(m.jobs[id].name)
		if err != nil {
			return nil, err
		}
		if state == stateNone || state == stateComplete {
			delete(m.jobs, id)
		}
	}
	return sortedJobIDs(m.jobs), nil
}

func (m *taskManager) runningServices() ([]string, error) {
	for _, name := range sortedNames(m.services) {
		state, err := m.sched.triggerState(name)
		if err != nil {
			return nil, err
		}
		if state == stateNone || state == stateComplete {
			delete(m.services, name)
		}
	}
	return sortedNames(m.services), nil
}

func (m *taskManager) newJobID() int64 {
	for {
		id := rand.Int64N(jobIDBound)
		if _, taken := m.jobs[id]; taken {
			continue
		}
		if m.sched != nil && m.sched.triggers[strconv.FormatInt(id, 10)] != nil {
			continue
		}
		return id
	}
}

func (m *taskManager) scheduleOneTime(job *jobDetail, timeMs int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.newJobID()
	s, err := m.getScheduler()
	if err != nil {
		return 0, err
	}
	spec := &triggerSpec{
		name:        strconv.FormatInt(id, 10),
		startMs:     timeMs,
		endMs:       noFireTime,
		repeatCount: 0,
		group:       triggerGroup,
		job:         job,
	}
	if err := s.scheduleJob(spec); err != nil {
		return 0, err
	}
	m.jobs[id] = spec
	return id, nil
}

type intervalSchedule struct {
	intervalMs    int64
	maxCount      int64
	startMs       int64
	endMs         int64
	waitingPolicy string
}

func (m *taskManager) buildIntervalSpec(name string, job *jobDetail, sched intervalSchedule) (*triggerSpec, error) {
	if sched.intervalMs < 0 {
		return nil, balError("Repeat interval must be >= 0")
	}
	repeatCount := int64(repeatIndefinitely)
	if sched.maxCount > 0 {
		repeatCount = sched.maxCount - 1
	}
	startMs := sched.startMs
	if startMs == noFireTime {
		startMs = m.nowMs()
	}
	if sched.endMs != noFireTime && startMs > sched.endMs {
		return nil, balError("End time cannot be before start time")
	}
	group := triggerGroup
	if sched.waitingPolicy == policyLogAndIgnore {
		group = logTriggerGroup
	}
	return &triggerSpec{
		name:          name,
		startMs:       startMs,
		endMs:         sched.endMs,
		intervalMs:    sched.intervalMs,
		repeatCount:   repeatCount,
		group:         group,
		ignoreMisfire: sched.waitingPolicy == policyWait,
		job:           job,
	}, nil
}

func (m *taskManager) scheduleInterval(job *jobDetail, sched intervalSchedule) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.newJobID()
	s, err := m.getScheduler()
	if err != nil {
		return 0, err
	}
	spec, err := m.buildIntervalSpec(strconv.FormatInt(id, 10), job, sched)
	if err != nil {
		return 0, err
	}
	if err := s.scheduleJob(spec); err != nil {
		return 0, err
	}
	m.jobs[id] = spec
	return id, nil
}

func (m *taskManager) jobSpec(id int64) (*triggerSpec, error) {
	spec, ok := m.jobs[id]
	if !ok {
		return nil, balError("Invalid job id: " + strconv.FormatInt(id, 10))
	}
	return spec, nil
}

func (m *taskManager) unscheduleJob(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	spec, err := m.jobSpec(id)
	if err != nil {
		return err
	}
	if err := m.sched.unscheduleJob(spec.name); err != nil {
		return err
	}
	running, err := m.runningJobs()
	if err != nil {
		return err
	}
	if len(running) == 0 {
		m.sched.shutdownNow()
	}
	return nil
}

func (m *taskManager) pauseJob(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	spec, err := m.jobSpec(id)
	if err != nil {
		return err
	}
	return m.sched.pauseJob(spec.name)
}

func (m *taskManager) resumeJob(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	spec, err := m.jobSpec(id)
	if err != nil {
		return err
	}
	return m.sched.resumeJob(spec.name)
}

func (m *taskManager) pauseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sched == nil {
		return errSchedulerShutdown
	}
	return m.sched.pauseAll()
}

func (m *taskManager) resumeAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sched == nil {
		return errSchedulerShutdown
	}
	return m.sched.resumeAll()
}

func (m *taskManager) getRunningJobs() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids, err := m.runningJobs()
	if err != nil {
		m.printMessage(m.pal.IO.Stderr, err.Error())
		return nil
	}
	return ids
}

func (m *taskManager) scheduleServices(names []string, specFor func(name string) (*triggerSpec, error)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.getScheduler()
	if err != nil {
		return err
	}
	for _, name := range names {
		spec, err := specFor(name)
		if err != nil {
			return err
		}
		if err := s.scheduleJob(spec); err != nil {
			return err
		}
		m.services[name] = spec
	}
	return nil
}

func (m *taskManager) unscheduleService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.services[name]; !ok {
		return nil
	}
	if err := m.sched.unscheduleJob(name); err != nil {
		return err
	}
	running, err := m.runningServices()
	if err != nil {
		return err
	}
	if len(running) == 0 {
		m.sched.shutdownNow()
	}
	return nil
}

func (m *taskManager) printMessage(write func([]byte) (int, error), msg string) {
	stamp := m.pal.Time.Now().UTC().Format("2006-01-02 15:04:05.000")
	_, _ = write([]byte("time = " + stamp + " message = " + msg + "\n"))
}

// logMisfire mirrors jBallerina's trigger listener: the job id is looked up
// among the scheduled jobs only, so listener services log a "null" id.
func (m *taskManager) logMisfire(t *trigger) {
	jobID := "null"
	for id, spec := range m.jobs {
		if spec == t.spec {
			jobID = strconv.FormatInt(id, 10)
			break
		}
	}
	startTime := time.UnixMilli(t.startMs).Format("Mon Jan 02 15:04:05 MST 2006")
	m.printMessage(m.pal.IO.Stderr, "The trigger for time["+startTime+
		"] has ignored as couldn't get the resources to execute the job["+jobID+"]")
}

// notifyFailure applies the job's error policy on the scheduler that fired it.
func (m *taskManager) notifyFailure(s *scheduler, spec *triggerSpec, msg string) {
	policy := spec.job.errorPolicy
	if policy == policyLogAndTerminate || policy == policyLogAndContinue {
		m.printMessage(m.pal.IO.Stdout, "Unable to execute the job["+spec.name+"]. "+msg)
	}
	if policy != policyLogAndTerminate && policy != policyTerminate {
		return
	}
	m.mu.Lock()
	err := s.unscheduleJob(spec.name)
	m.mu.Unlock()
	if err != nil && policy == policyLogAndTerminate {
		m.printMessage(m.pal.IO.Stdout, "org.quartz.SchedulerException: "+err.Error())
	}
}

func sortedJobIDs(jobs map[int64]*triggerSpec) []int64 {
	ids := make([]int64, 0, len(jobs))
	for id := range jobs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedNames(services map[string]*triggerSpec) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
