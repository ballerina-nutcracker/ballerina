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
	"regexp"
	"strings"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const listenerStateField = "$ftpListener"

type ageFilter struct {
	minAge   *float64
	maxAge   *float64
	creation bool
}

type dependencyCondition struct {
	target        *regexp.Regexp
	requiredFiles []string
	mode          string
	requiredCount int64
}

// watchConfig is what a consumer filters a directory by: the @ftp:ServiceConfig of a service or,
// without one, the deprecated listener-level fields.
type watchConfig struct {
	path         string
	pattern      *regexp.Regexp
	age          *ageFilter
	dependencies []dependencyCondition
}

type listenerState struct {
	mu        sync.Mutex
	config    *values.Map
	ep        *endpoint
	retry     *retrySettings
	lax       bool
	legacy    watchConfig
	modeSet   bool
	multiPath bool
	services  []*serviceContext
	typeKeys  map[string]bool
	legacyCon *consumer
	shared    *values.Object
	polling   bool
}

func listenerStateOf(obj *values.Object) *listenerState {
	v, _ := obj.Get(listenerStateField)
	state, _ := v.(*listenerState)
	return state
}

func (m *ftpModule) registerListenerExterns() {
	register := func(name string, fn extern.NativeFunc) {
		runtime.RegisterExternFunction(m.rt, orgName, moduleName, name, fn)
	}
	register("initListener", m.initListener)
	register("register", m.register)
	register("deregister", m.deregister)
	register("poll", m.poll)
	register("cleanup", m.cleanup)
}

// compilePattern compiles a pattern for whole-string matching, as Java's String.matches uses it.
func compilePattern(pattern string) (*regexp.Regexp, error) {
	if _, err := regexp.Compile(pattern); err != nil {
		return nil, err
	}
	return regexp.Compile(`^(?:` + pattern + `)$`)
}

func optionalFloat(m *values.Map, key string) *float64 {
	v, ok := m.Get(key)
	if !ok || v == nil {
		return nil
	}
	f, _ := values.NumericConvertToFloat(v)
	return &f
}

func parseAgeFilter(m *values.Map) (*ageFilter, error) {
	if m == nil {
		return nil, nil
	}
	mode, _ := getString(m, "ageCalculationMode")
	f := &ageFilter{minAge: optionalFloat(m, "minAge"), maxAge: optionalFloat(m, "maxAge"), creation: mode == "CREATION_TIME"}
	if f.minAge != nil && *f.minAge < 0 {
		return nil, newError(kindInvalidConfig, "fileAgeFilter.minAge must be positive or zero (got: "+values.FormatFloat(*f.minAge)+")")
	}
	if f.maxAge != nil && *f.maxAge < 0 {
		return nil, newError(kindInvalidConfig, "fileAgeFilter.maxAge must be positive or zero (got: "+values.FormatFloat(*f.maxAge)+")")
	}
	if f.minAge != nil && f.maxAge != nil && *f.minAge > *f.maxAge {
		return nil, newError(kindInvalidConfig, "fileAgeFilter.minAge ("+values.FormatFloat(*f.minAge)+
			") must not exceed fileAgeFilter.maxAge ("+values.FormatFloat(*f.maxAge)+")")
	}
	return f, nil
}

// parseDependencies validates the dependency patterns; describe builds the error message for an
// invalid pattern at a position.
func parseDependencies(list *values.List, describe func(i int, field string, pattern string, err error) string) ([]dependencyCondition, error) {
	if list == nil {
		return nil, nil
	}
	conditions := make([]dependencyCondition, 0, list.Len())
	for i := 0; i < list.Len(); i++ {
		m := list.Get(i).(*values.Map)
		targetPattern, _ := getString(m, "targetPattern")
		target, err := compilePattern(targetPattern)
		if err != nil {
			return nil, newError(kindInvalidConfig, describe(i, "targetPattern", targetPattern, err))
		}
		required := getStrings(m, "requiredFiles")
		for j, r := range required {
			if _, err := compilePattern(r); err != nil {
				return nil, newError(kindInvalidConfig, describe(i, "requiredFiles["+formatInt(int64(j))+"]", r, err))
			}
		}
		mode, ok := getString(m, "matchingMode")
		if !ok {
			mode = "ALL"
		}
		count := int64(1)
		if v, ok := m.Get("requiredFileCount"); ok {
			count, _ = v.(int64)
		}
		conditions = append(conditions, dependencyCondition{target: target, requiredFiles: required, mode: mode, requiredCount: count})
	}
	return conditions, nil
}

func listValue(m *values.Map, key string) *values.List {
	v, _ := m.Get(key)
	l, _ := v.(*values.List)
	return l
}

func (m *ftpModule) initListener(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	obj := args[0].(*values.Object)
	config := args[1].(*values.Map)
	if auth := getMap(config, "auth"); auth != nil {
		if credentials := getMap(auth, "credentials"); credentials != nil {
			if username, _ := getString(credentials, "username"); strings.TrimSpace(username) == "" {
				return m.errorResult(ctx, newError(kindError, "Username cannot be empty"))
			}
		}
	}
	state := &listenerState{config: config, lax: getBool(config, "laxDataBinding"), typeKeys: map[string]bool{}}
	state.legacy.path, _ = getString(config, "path")
	if pattern, ok := getString(config, "fileNamePattern"); ok {
		re, err := compilePattern(pattern)
		if err != nil {
			return m.errorResult(ctx, newError(kindInvalidConfig, "Invalid regex pattern for fileNamePattern: "+err.Error()))
		}
		state.legacy.pattern = re
	}
	platform := m.rt.Platform()
	ep, err := parseAuth(config, platform, listenerMessages)
	if err != nil {
		return m.errorResult(ctx, err)
	}
	if state.legacy.age, err = parseAgeFilter(getMap(config, "fileAgeFilter")); err != nil {
		return m.errorResult(ctx, err)
	}
	if err := parseTransport(config, ep); err != nil {
		return m.errorResult(ctx, err)
	}
	if state.retry, err = validateRetryConfig(config); err != nil {
		return m.errorResult(ctx, err)
	}
	state.ep = ep
	obj.Put(listenerStateField, state)
	return nil, nil
}

func serviceConfigOf(ctx *extern.Context, svc *values.Object) *values.Map {
	annotations, _ := ctx.ObjectAnnotations(svc)
	for key, value := range annotations {
		if strings.HasPrefix(key, orgName+"/"+moduleName+":") && strings.HasSuffix(key, ":ServiceConfig") {
			config, _ := value.(*values.Map)
			return config
		}
	}
	return nil
}

func serviceTypeKey(svc *values.Object) string {
	for _, name := range svc.RemoteMethodNames() {
		if key, ok := svc.MethodLookupKey(model.RemoteMethodName(name)); ok {
			if i := strings.LastIndexByte(key, '.'); i > 0 {
				return key[:i]
			}
		}
	}
	return ""
}

func normalizeServicePath(p string) string {
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

func (m *ftpModule) serviceWatchConfig(config *values.Map) (watchConfig, error) {
	var w watchConfig
	p, _ := getString(config, "path")
	if strings.TrimSpace(p) == "" {
		return w, newError(kindInvalidConfig, "The 'path' field cannot be empty in @ftp:ServiceConfig annotation.")
	}
	w.path = normalizeServicePath(p)
	if pattern, ok := getString(config, "fileNamePattern"); ok {
		re, err := compilePattern(pattern)
		if err != nil {
			return w, newError(kindInvalidConfig, "Invalid regex pattern '"+pattern+"' in @ftp:ServiceConfig.fileNamePattern: "+err.Error())
		}
		w.pattern = re
	}
	deps, err := parseDependencies(listValue(config, "fileDependencyConditions"), func(_ int, field, pattern string, err error) string {
		if strings.HasPrefix(field, "requiredFiles") {
			field = "requiredFiles"
		}
		return "Invalid regex pattern '" + pattern + "' in fileDependencyConditions." + field + ": " + err.Error()
	})
	if err != nil {
		return w, err
	}
	w.dependencies = deps
	if w.age, err = parseAgeFilter(getMap(config, "fileAgeFilter")); err != nil {
		return w, err
	}
	return w, nil
}

func (m *ftpModule) register(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	state := listenerStateOf(args[0].(*values.Object))
	svc := args[1].(*values.Object)
	if state == nil {
		return m.errorResult(ctx, newError(kindError, "FTP listener is not initialized."))
	}
	handlers, err := resolveHandlers(ctx, m.types, svc)
	if err != nil {
		return m.errorResult(ctx, err)
	}
	serviceConfig := serviceConfigOf(ctx, svc)
	state.mu.Lock()
	defer state.mu.Unlock()
	if !state.modeSet {
		state.modeSet = true
		state.multiPath = serviceConfig != nil
	} else if state.multiPath && serviceConfig == nil {
		return m.errorResult(ctx, newError(kindInvalidConfig, "All services attached to a listener must use "+
			"@ftp:ServiceConfig annotation when any service uses it. The new service is missing the annotation."))
	} else if !state.multiPath && serviceConfig != nil {
		return m.errorResult(ctx, newError(kindInvalidConfig, "All services attached to a listener must use "+
			"@ftp:ServiceConfig annotation when any service uses it. Previous services did not use the annotation."))
	}
	typeKey := serviceTypeKey(svc)
	if state.typeKeys[typeKey] {
		return m.errorResult(ctx, newError(kindError, "Failed to register service."))
	}
	sc := &serviceContext{svc: svc, handlers: handlers}
	if state.multiPath {
		watch, err := m.serviceWatchConfig(serviceConfig)
		if err != nil {
			return m.errorResult(ctx, err)
		}
		for _, other := range state.services {
			if other.consumer.watch.path == watch.path {
				return m.errorResult(ctx, newError(kindInvalidConfig, "Duplicate path '"+watch.path+
					"' in @ftp:ServiceConfig. Each service must monitor a unique path."))
			}
		}
		con, err := m.newConsumer(state, watch, "Unable to initialize the connection with the server. ")
		if err != nil {
			return m.errorResult(ctx, err)
		}
		sc.consumer = con
	} else {
		deps, err := parseDependencies(listValue(state.config, "fileDependencyConditions"), func(i int, field, _ string, err error) string {
			return "Invalid regex pattern for fileDependencyConditions[" + formatInt(int64(i)) + "]." + field + ": " + err.Error()
		})
		if err != nil {
			return m.errorResult(ctx, err)
		}
		if state.legacyCon == nil {
			watch := state.legacy
			watch.dependencies = deps
			con, err := m.newConsumer(state, watch, "Failed to initialize File server connector. ")
			if err != nil {
				return m.errorResult(ctx, err)
			}
			state.legacyCon = con
		}
		sc.consumer = state.legacyCon
	}
	if handlers.needsCaller() {
		caller, err := m.callerFor(ctx, state)
		if err != nil {
			return m.errorResult(ctx, err)
		}
		sc.caller = caller
	}
	state.typeKeys[typeKey] = true
	state.services = append(state.services, sc)
	return nil, nil
}

func (m *ftpModule) callerFor(ctx *extern.Context, state *listenerState) (*values.Object, error) {
	if !state.multiPath && state.shared != nil {
		return state.shared, nil
	}
	handle, ok := ctx.LookupFunction(orgName, moduleName, "createCaller")
	if !ok {
		return nil, newError(kindError, "Failed to create the caller")
	}
	result, err := ctx.InvokeFunction(handle, []values.BalValue{state.config})
	if err != nil {
		return nil, err
	}
	if balErr, isErr := result.(*values.Error); isErr {
		return nil, &ftpError{kind: kindError, message: balErr.Message, cause: &balCause{value: balErr}}
	}
	caller := result.(*values.Object)
	if !state.multiPath {
		state.shared = caller
	}
	return caller, nil
}

func (m *ftpModule) deregister(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
	return nil, nil
}

func (m *ftpModule) cleanup(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	state := listenerStateOf(args[0].(*values.Object))
	if state == nil {
		return nil, nil
	}
	state.mu.Lock()
	services := state.services
	state.services = nil
	state.legacyCon = nil
	state.shared = nil
	state.modeSet = false
	state.typeKeys = map[string]bool{}
	state.mu.Unlock()
	closed := map[*values.Object]bool{}
	consumers := map[*consumer]bool{}
	var errs []error
	for _, sc := range services {
		consumers[sc.consumer] = true
		if sc.caller == nil || closed[sc.caller] {
			continue
		}
		closed[sc.caller] = true
		if err := closeCaller(ctx, sc.caller); err != nil {
			errs = append(errs, err)
		}
	}
	for con := range consumers {
		_ = con.remote.close()
	}
	if len(errs) > 0 {
		return m.errorResult(ctx, newError(kindError, "Error occurred while closing the caller: "+errs[0].Error()))
	}
	return nil, nil
}

func closeCaller(ctx *extern.Context, caller *values.Object) error {
	handle, ok := ctx.LookupFunction(orgName, moduleName, "closeCaller")
	if !ok {
		return nil
	}
	result, err := ctx.InvokeFunction(handle, []values.BalValue{caller})
	if err != nil {
		return err
	}
	if balErr, isErr := result.(*values.Error); isErr {
		return errors.New(balErr.Message)
	}
	return nil
}

func (m *ftpModule) poll(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	state := listenerStateOf(args[0].(*values.Object))
	if state == nil {
		return m.errorResult(ctx, newError(kindError, "FTP listener is not initialized. Attach a service before polling."))
	}
	state.mu.Lock()
	if len(state.services) == 0 {
		state.mu.Unlock()
		return m.errorResult(ctx, newError(kindError, "FTP listener is not initialized. Attach a service before polling."))
	}
	if state.polling {
		state.mu.Unlock()
		return nil, nil
	}
	state.polling = true
	services := state.services
	multiPath := state.multiPath
	state.mu.Unlock()
	defer func() {
		state.mu.Lock()
		state.polling = false
		state.mu.Unlock()
	}()
	if multiPath {
		for _, sc := range services {
			if event, ok := sc.consumer.consume(); ok {
				m.dispatch(ctx, state, sc, event)
			}
		}
		return nil, nil
	}
	if event, ok := services[0].consumer.consume(); ok {
		for _, sc := range services {
			m.dispatch(ctx, state, sc, event)
		}
	}
	return nil, nil
}
