// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
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

package exec

import (
	"strconv"
	"strings"

	"ballerina-lang-go/decimal"
	"ballerina-lang-go/model"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/runtime/internal/modules"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

// LookupObjectMethod resolves a regular method named methodName on obj. The
// second return is false if obj has no such method. Remote methods are not
// resolved through this entry point; use LookupRemoteMethod for those.
func LookupObjectMethod(ctx *extern.Context, obj *values.Object, methodName string) (any, bool) {
	return lookupByMethodName(ctx, obj, methodName)
}

// LookupRemoteMethod resolves the remote method named methodName on obj.
// The second return is false if obj has no such remote method. Callers pass
// the declared method name not the mangled method name;
func LookupRemoteMethod(ctx *extern.Context, obj *values.Object, methodName string) (any, bool) {
	return lookupByMethodName(ctx, obj, model.RemoteMethodName(methodName))
}

func LookupFunction(env *extern.Env, org, module, name string) (any, bool) {
	reg := env.Registry.(*modules.Registry)
	key := org + "/" + module + ":" + name
	if fn := reg.GetBIRFunction(key); fn != nil {
		return NewBIRHandle(fn), true
	}
	if ef := reg.GetNativeFunction(key); ef != nil {
		return NewNativeHandle(ef.Impl), true
	}
	return nil, false
}

// LookupResourceMethod resolves a resource method named resourceMethodName
// on obj. The second return is false if no candidate matches or if more
// than one candidate matches (ambiguous dispatch).
//
// path contains a value for every path segment of the source-level
// resource access expression (literal AND computed). The matcher compares
// each value's shape against the candidate's segment types; the invoked
// function only receives the computed-segment values plus the rest list,
// as constructed by buildResourceCallArgs.
func LookupResourceMethod(ctx *extern.Context, obj *values.Object, resourceMethodName string, path []values.BalValue) (any, bool) {
	matches := resourceFnCandidates(ctx, obj, resourceMethodName, path)
	if len(matches) != 1 {
		return nil, false
	}
	return newResourceHandle(obj, matches[0], path), true
}

// LookupResourceMethodByPath resolves a resource method from a RAW, untyped
// path: the URL-style string segments relative to the receiver's attach point
// (e.g. ["items", "42"]). Unlike LookupResourceMethod — which matches the
// shapes of already-typed BalValues — this entry point coerces each string
// segment to the parameter type the candidate declares (int/float/decimal/
// boolean/string, plus literal segments), then selects the unique matching
// candidate. It is meant for network dispatchers (e.g. the HTTP listener) that
// only have the wire-format path.
//
// The second result is the number of NON-path parameters the resolved resource
// expects (a value injected by the caller, such as an http:Request); path
// parameters are already baked into the returned handle. The third result is
// false when no candidate matches or when more than one matches (ambiguous).
func LookupResourceMethodByPath(ctx *extern.Context, obj *values.Object, accessor string, segments []string) (any, int, bool) {
	candidates, ok := obj.ResourceEntries(accessor)
	if !ok {
		return nil, 0, false
	}
	var (
		matchEntry *values.ResourceEntry
		matchPath  []values.BalValue
	)
	for i := range candidates {
		pathVals, ok := coercePathForEntry(ctx.TypeCtx, &candidates[i], segments)
		if !ok {
			continue
		}
		if matchEntry != nil {
			// Ambiguous: more than one candidate matches.
			return nil, 0, false
		}
		matchEntry = &candidates[i]
		matchPath = pathVals
	}
	if matchEntry == nil {
		return nil, 0, false
	}
	return newResourceHandle(obj, matchEntry, matchPath), resourceExtraArgCount(ctx, matchEntry), true
}

// resourceExtraArgCount returns how many parameters of the resource function
// are not bound from the path (i.e. supplied by the caller). It mirrors the
// arity accounting the path matcher relies on: total required params minus the
// path-bound segments. A rest path segment, if any, is folded into
// RequiredParams as a single slot (see transformResourceMethodInner), so it
// must be counted as path-bound here too, alongside the non-literal fixed
// segments.
func resourceExtraArgCount(ctx *extern.Context, entry *values.ResourceEntry) int {
	fn := ctx.Env.Registry.(*modules.Registry).GetBIRFunction(entry.FunctionLookupKey)
	if fn == nil {
		return 0
	}
	nonLiteral := 0
	for i := range entry.PathSegments {
		if _, isLit := values.LiteralPathSegment(entry.PathSegments[i]); !isLit {
			nonLiteral++
		}
	}
	if !semtypes.IsNever(entry.RestSegmentTy) {
		nonLiteral++
	}
	if extra := len(fn.RequiredParams) - nonLiteral; extra > 0 {
		return extra
	}
	return 0
}

// coercePathForEntry coerces the URL string segments to the typed values the
// candidate resource entry expects, including any rest segments. Returns
// (nil, false) when the segment count or any segment type does not match.
func coercePathForEntry(tc semtypes.Context, entry *values.ResourceEntry, segments []string) ([]values.BalValue, bool) {
	required := len(entry.PathSegments)
	if len(segments) < required {
		return nil, false
	}
	if len(segments) > required && semtypes.IsNever(entry.RestSegmentTy) {
		return nil, false
	}
	result := make([]values.BalValue, len(segments))
	for i := 0; i < required; i++ {
		v, ok := coerceSegment(tc, entry.PathSegments[i].Ty, segments[i])
		if !ok {
			return nil, false
		}
		result[i] = v
	}
	for i := required; i < len(segments); i++ {
		v, ok := coerceSegment(tc, entry.RestSegmentTy, segments[i])
		if !ok {
			return nil, false
		}
		result[i] = v
	}
	return result, true
}

// coerceSegment coerces a single URL path segment string to a typed value
// matching segTy. Literal segments must equal the stored literal (after
// decoding Ballerina quoted-identifier syntax). Parameter segments are parsed
// by type; an unrecognised type is accepted as a string.
func coerceSegment(tc semtypes.Context, segTy semtypes.SemType, s string) (values.BalValue, bool) {
	if shape := semtypes.SingleShape(segTy); shape.IsPresent() {
		if lit, ok := shape.Get().Value.(string); ok {
			if s != decodeBalIdentifier(lit) {
				return nil, false
			}
			// s is already the decoded literal text (equal to decodeBalIdentifier(lit));
			// return it rather than the raw, possibly escaped source token.
			return s, true
		}
	}
	if semtypes.IsSubtype(tc, semtypes.INT, segTy) {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n, true
		}
	}
	if semtypes.IsSubtype(tc, semtypes.FLOAT, segTy) {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
	}
	if semtypes.IsSubtype(tc, semtypes.DECIMAL, segTy) {
		if d, err := decimal.FromString(s); err == nil {
			return d, true
		}
	}
	if semtypes.IsSubtype(tc, semtypes.BOOLEAN, segTy) {
		// Matches lang.boolean:fromString's accepted range for consistency
		// with the other typed path parameters.
		switch strings.ToLower(s) {
		case "true", "1":
			return true, true
		case "false", "0":
			return false, true
		}
	}
	if semtypes.IsSubtype(tc, semtypes.STRING, segTy) {
		return s, true
	}
	return nil, false
}

// decodeBalIdentifier converts a Ballerina identifier token text to its
// URL-path form: it strips a leading quoted-identifier prefix (') and removes
// backslash escapes (\X -> X).
func decodeBalIdentifier(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[0] == '\'' {
		s = s[1:]
	}
	if !containsByte(s, '\\') {
		return s
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
		}
		out = append(out, s[i])
	}
	return string(out)
}

func containsByte(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}

// Invoke calls the closure captured by the handle returned from one of
// the Lookup* functions.
func Invoke(ctx *extern.Context, h any, args []values.BalValue) (values.BalValue, error) {
	return h.(*InvokableHandle).invoke(ctx, args)
}

func lookupByMethodName(ctx *extern.Context, obj *values.Object, methodName string) (any, bool) {
	lookupKey, found := obj.MethodLookupKey(methodName)
	if !found {
		return nil, false
	}
	return lookupByKey(ctx, lookupKey)
}

func lookupByKey(ctx *extern.Context, lookupKey string) (any, bool) {
	reg := ctx.Env.Registry.(*modules.Registry)
	if fn := reg.GetBIRFunction(lookupKey); fn != nil {
		return NewBIRHandle(fn), true
	}
	if externFn := reg.GetNativeFunction(lookupKey); externFn != nil {
		return NewNativeHandle(externFn.Impl), true
	}
	return nil, false
}
