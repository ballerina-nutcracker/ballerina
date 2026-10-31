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
	"fmt"
	"path"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type paramKind int

const (
	paramContent paramKind = iota
	paramFileInfo
	paramCaller
)

type handler struct {
	name     string
	params   []paramKind
	firstTy  semtypes.SemType
	isStream bool
}

func (h *handler) hasCaller() bool {
	for _, p := range h.params {
		if p == paramCaller {
			return true
		}
	}
	return false
}

type handlerSet struct {
	content      map[string]*handler
	onDelete     *handler
	deleteArray  bool
	onFileChange *handler
	onError      *handler
}

func (h *handlerSet) needsCaller() bool {
	for _, c := range h.content {
		if c.hasCaller() {
			return true
		}
	}
	for _, c := range []*handler{h.onDelete, h.onFileChange, h.onError} {
		if c != nil && c.hasCaller() {
			return true
		}
	}
	return false
}

var contentMethods = map[string]bool{"onFile": true, "onFileText": true, "onFileJson": true, "onFileXml": true, "onFileCsv": true}

const allowedHandlers = "onFile, onFileText, onFileJson, onFileXml, onFileCsv (format-specific) or onFileDelete."

func describeMethod(ctx *extern.Context, svc *values.Object, name string) *handler {
	tc := ctx.TypeCtx()
	methodTy := semtypes.ObjectMemberType(tc, semtypes.StringConst(model.RemoteMethodName(name)), svc.Type)
	paramList := semtypes.FunctionParamListType(tc, methodTy)
	h := &handler{name: name}
	for i := 0; ; i++ {
		paramTy := semtypes.ListMemberTypeInnerVal(tc, paramList, semtypes.IntConst(int64(i)))
		if semtypes.IsNever(paramTy) {
			break
		}
		switch {
		case i == 0:
			h.firstTy = paramTy
			h.isStream = semtypes.IsSubtypeSimple(paramTy, semtypes.Stream)
			if semtypes.IsSubtypeSimple(paramTy, semtypes.Object) {
				h.params = append(h.params, paramCaller)
			} else {
				h.params = append(h.params, paramContent)
			}
		case semtypes.IsSubtypeSimple(paramTy, semtypes.Object):
			h.params = append(h.params, paramCaller)
		default:
			h.params = append(h.params, paramFileInfo)
		}
	}
	return h
}

// resolveHandlers stands in for the jBallerina compiler plugin, which rejects these service
// shapes at compile time; here they fail when the service is attached.
func resolveHandlers(ctx *extern.Context, types *ftpTypes, svc *values.Object) (*handlerSet, error) {
	if len(svc.AllResourceMethodNames()) > 0 {
		return nil, newError(kindError, "Resource functions are not allowed for ftp services.")
	}
	set := &handlerSet{content: map[string]*handler{}}
	var onFileDeleted *handler
	for _, name := range svc.RemoteMethodNames() {
		h := describeMethod(ctx, svc, name)
		switch {
		case contentMethods[name]:
			if len(h.params) > 3 {
				return nil, newError(kindError, "Too many parameters for '"+name+"'. Format-specific handlers accept at most 3 parameters: (content, fileInfo?, caller?).")
			}
			if len(h.params) == 0 {
				return nil, newError(kindError, "Mandatory parameter missing for '"+name+"'.")
			}
			if err := checkSupportedContent(ctx, types, h); err != nil {
				return nil, err
			}
			set.content[name] = h
		case name == "onFileDelete":
			if len(h.params) > 2 {
				return nil, newError(kindError, "Too many parameters for onFileDelete. Accepts at most 2 parameters: (deletedFile, caller?).")
			}
			set.onDelete = h
		case name == "onFileDeleted":
			if len(h.params) > 2 {
				return nil, newError(kindError, "Too many parameters for onFileDeleted. Accepts at most 2 parameters: (deletedFiles, caller?).")
			}
			onFileDeleted = h
		case name == "onFileChange":
			if len(h.params) > 2 {
				return nil, newError(kindError, "Too many parameters. onFileChange accepts at most 2 parameters: (WatchEvent, Caller?).")
			}
			set.onFileChange = h
		case name == "onError":
			if len(h.params) > 2 {
				return nil, newError(kindError, "Too many parameters for onError. Accepts at most 2 parameters: (error, caller?).")
			}
			set.onError = h
		default:
			return nil, newError(kindError, "Invalid remote method. Allowed handlers: "+allowedHandlers)
		}
	}
	if set.onDelete != nil && onFileDeleted != nil {
		return nil, newError(kindError, "Cannot use both 'onFileDelete' and 'onFileDeleted' methods. Use only 'onFileDelete' as 'onFileDeleted' is deprecated.")
	}
	if onFileDeleted != nil {
		set.onDelete, set.deleteArray = onFileDeleted, true
	}
	if set.onFileChange != nil && (len(set.content) > 0 || set.onDelete != nil) {
		return nil, newError(kindError, "Cannot mix event-based handler (onFileChange) with format-specific handlers "+
			"(onFile, onFileText, onFileJson, onFileXml, onFileCsv, onFileDelete).")
	}
	if set.onFileChange == nil && len(set.content) == 0 && set.onDelete == nil {
		return nil, newError(kindError, "Service must define at least one handler method: "+allowedHandlers)
	}
	return set, nil
}

func checkSupportedContent(ctx *extern.Context, types *ftpTypes, h *handler) error {
	switch h.name {
	case "onFileCsv":
		return newError(kindError, "Handling CSV content with onFileCsv is not yet supported")
	case "onFileXml":
		if semtypes.IsNever(semtypes.Intersect(h.firstTy, semtypes.XML)) {
			return newError(kindError, "Binding XML content to a record type is not yet supported")
		}
	case "onFile":
		if !h.isStream && !semtypes.IsSubtype(ctx.TypeCtx(), types.byteArrTy, h.firstTy) {
			return newError(kindError, "Invalid parameter type for handler 'onFile'. Expected 'byte[]' or 'stream<byte[], error?>'.")
		}
	}
	return nil
}

type serviceContext struct {
	svc      *values.Object
	handlers *handlerSet
	consumer *consumer
	caller   *values.Object
}

func (m *ftpModule) logError(message string) {
	_, _ = m.rt.Platform().IO.Stderr([]byte("error: " + message + "\n"))
}

// start runs a handler on its own strand without waiting for it, as jBallerina does; a returned
// error or a panic is printed to stderr and the listener keeps polling.
func (m *ftpModule) start(ctx *extern.Context, sc *serviceContext, name string, args []values.BalValue) {
	handle, ok := ctx.LookupRemoteMethod(sc.svc, name)
	if !ok {
		return
	}
	go m.invokeHandler(handle, append([]values.BalValue{sc.svc}, args...))
}

func (m *ftpModule) invokeHandler(handle extern.MethodHandle, args []values.BalValue) {
	ctx := m.rt.AcquirePooledContext()
	defer m.rt.ReleasePooledContext(ctx)
	defer func() {
		if rec := recover(); rec != nil {
			ctx.ReleaseAllHeldLocks()
			m.logError(panicMessage(rec))
		}
	}()
	result, err := ctx.InvokeMethod(handle, args)
	if err != nil {
		m.logError(err.Error())
	} else if balErr, isErr := result.(*values.Error); isErr {
		m.logError(balErr.Message)
	}
}

func panicMessage(r any) string {
	switch v := r.(type) {
	case *values.Error:
		return v.Message
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", r)
	}
}

func (m *ftpModule) handlerArgs(h *handler, sc *serviceContext, content, fileInfo values.BalValue) []values.BalValue {
	args := make([]values.BalValue, len(h.params))
	for i, kind := range h.params {
		switch kind {
		case paramContent:
			args[i] = content
		case paramFileInfo:
			args[i] = fileInfo
		case paramCaller:
			args[i] = sc.caller
		}
	}
	return args
}

func (m *ftpModule) dispatch(ctx *extern.Context, state *listenerState, sc *serviceContext, event fileEvent) {
	m.types.resolveRecordTypes(ctx)
	h := sc.handlers
	switch {
	case len(h.content) > 0:
		for _, f := range event.added {
			m.dispatchContent(ctx, state, sc, f)
		}
		m.dispatchDeletes(ctx, sc, event.deleted)
	case h.onDelete != nil:
		m.dispatchDeletes(ctx, sc, event.deleted)
	case h.onFileChange != nil:
		m.dispatchChange(ctx, sc, event)
	}
}

func (m *ftpModule) dispatchDeletes(ctx *extern.Context, sc *serviceContext, deleted []string) {
	h := sc.handlers.onDelete
	if h == nil || len(deleted) == 0 {
		return
	}
	if sc.handlers.deleteArray {
		m.start(ctx, sc, h.name, m.handlerArgs(h, sc, m.types.strings(deleted), nil))
		return
	}
	for _, d := range deleted {
		m.start(ctx, sc, h.name, m.handlerArgs(h, sc, d, nil))
	}
}

func (m *ftpModule) dispatchChange(ctx *extern.Context, sc *serviceContext, event fileEvent) {
	tc := ctx.TypeCtx()
	ep := sc.consumer.remote.ep
	added := make([]values.BalValue, len(event.added))
	for i, f := range event.added {
		added[i] = listenerFileInfo(m.types, ep, f.logical, f.entry)
	}
	var watchEvent values.BalValue = values.NewMap(m.types.watchEventTy, m.types.watchEvAtom, false, []values.MapEntry{
		{Key: "addedFiles", Value: m.types.fileInfoList(added)},
		{Key: "deletedFiles", Value: m.types.strings(event.deleted)},
	})
	h := sc.handlers.onFileChange
	for i, kind := range h.params {
		if kind == paramCaller {
			continue
		}
		paramTy := semtypes.ListMemberTypeInnerVal(tc, semtypes.FunctionParamListType(tc,
			semtypes.ObjectMemberType(tc, semtypes.StringConst(model.RemoteMethodName(h.name)), sc.svc.Type)),
			semtypes.IntConst(int64(i)))
		if !semtypes.IsSubtype(tc, m.types.watchEventTy, paramTy) {
			if readonlyEvent, err := values.CloneWithType(tc, watchEvent, paramTy); err == nil {
				watchEvent = readonlyEvent
			}
		}
	}
	args := make([]values.BalValue, len(h.params))
	for i, kind := range h.params {
		if kind == paramCaller {
			args[i] = sc.caller
		} else {
			args[i] = watchEvent
		}
	}
	m.start(ctx, sc, h.name, args)
}

// contentHandlerFor routes a file by extension; a file whose format handler is not declared
// goes to onFile, and is skipped when there is none.
func contentHandlerFor(set *handlerSet, name string) *handler {
	method := "onFile"
	switch strings.ToLower(extension(name)) {
	case "json":
		method = "onFileJson"
	case "xml":
		method = "onFileXml"
	case "csv":
		method = "onFileCsv"
	case "txt", "log", "md":
		method = "onFileText"
	}
	if h, ok := set.content[method]; ok {
		return h
	}
	return set.content["onFile"]
}

func (m *ftpModule) dispatchContent(ctx *extern.Context, state *listenerState, sc *serviceContext, f addedFile) {
	h := contentHandlerFor(sc.handlers, path.Base(f.logical))
	if h == nil {
		return
	}
	tc := ctx.TypeCtx()
	remote := sc.consumer.remote
	fileURI := vfsURI(remote.ep, f.logical, false)
	var content values.BalValue
	if h.isStream {
		r, err := remote.openFile(f.logical)
		if err != nil {
			m.logError(err.Error())
			return
		}
		content = m.contentStream(r)
	} else {
		data, err := withRetry(ctx, state.retry, "fetchContent", func() ([]byte, error) {
			return remote.readAll(f.logical)
		})
		if err != nil {
			m.logError(err.Error())
			return
		}
		content, err = m.convertContent(tc, h, data, fileURI, state.lax)
		if err != nil {
			m.handleBindingError(ctx, sc, err)
			return
		}
	}
	fileInfo := listenerFileInfo(m.types, remote.ep, f.logical, f.entry)
	m.start(ctx, sc, h.name, m.handlerArgs(h, sc, content, fileInfo))
}

func (m *ftpModule) convertContent(tc semtypes.Context, h *handler, data []byte, fileURI string, lax bool) (values.BalValue, error) {
	switch h.name {
	case "onFileText":
		return decodeUTF8(data), nil
	case "onFileJson":
		return bindJSON(tc, m.types, data, h.firstTy, fileURI, lax)
	case "onFileXml":
		return bindXML(tc, data, h.firstTy, fileURI)
	}
	return m.types.bytes(data), nil
}

func (m *ftpModule) handleBindingError(ctx *extern.Context, sc *serviceContext, err error) {
	typed := asFtpError(err)
	onError := sc.handlers.onError
	if typed.kind != kindContentBinding || onError == nil {
		m.logError(typed.message)
		return
	}
	balErr := m.toBal(ctx, typed)
	args := make([]values.BalValue, len(onError.params))
	for i, kind := range onError.params {
		if kind == paramCaller {
			args[i] = sc.caller
		} else {
			args[i] = balErr
		}
	}
	m.start(ctx, sc, onError.name, args)
}
