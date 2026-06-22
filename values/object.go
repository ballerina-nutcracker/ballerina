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

package values

import (
	"strings"

	"ballerina-lang-go/semtypes"
)

type Object struct {
	Type       semtypes.SemType
	fields     map[string]BalValue
	methodKeys map[string]string
	rtable     map[string][]ResourceEntry
}

type ResourceEntry struct {
	PathSegments      []ResourcePathSegmentDef
	RestSegmentTy     semtypes.SemType
	FunctionLookupKey string
}

type ResourcePathSegmentDef struct {
	// Ty is a singleton string type for literal path segments
	// and the parameter type for path-parameter segments.
	Ty semtypes.SemType
}

// LiteralPathSegment returns the literal string of seg and true if seg is a
// literal path segment, otherwise it returns false.
func LiteralPathSegment(seg ResourcePathSegmentDef) (string, bool) {
	shape := semtypes.SingleShape(seg.Ty)
	if !shape.IsPresent() {
		return "", false
	}
	s, ok := shape.Get().Value.(string)
	return s, ok
}

func NewObject(typ semtypes.SemType, fieldValues map[string]BalValue, methodKeys map[string]string, rtable map[string][]ResourceEntry) *Object {
	if fieldValues == nil {
		fieldValues = make(map[string]BalValue)
	}
	if methodKeys == nil {
		methodKeys = make(map[string]string)
	}
	if rtable == nil {
		rtable = make(map[string][]ResourceEntry)
	}
	return &Object{
		Type:       typ,
		fields:     fieldValues,
		methodKeys: methodKeys,
		rtable:     rtable,
	}
}

func (o *Object) ResourceEntries(methodName string) ([]ResourceEntry, bool) {
	entries, ok := o.rtable[methodName]
	return entries, ok
}

// AllResourceMethodNames returns the accessor names (e.g. HTTP methods and
// "default") for which this object declares at least one resource method.
// The order is unspecified.
func (o *Object) AllResourceMethodNames() []string {
	names := make([]string, 0, len(o.rtable))
	for name := range o.rtable {
		names = append(names, name)
	}
	return names
}

func (o *Object) Put(field string, value BalValue) {
	o.fields[field] = value
}

func (o *Object) Get(field string) (BalValue, bool) {
	value, ok := o.fields[field]
	return value, ok
}

func (o *Object) MethodLookupKey(name string) (string, bool) {
	key, ok := o.methodKeys[name]
	return key, ok
}

// HasRemoteMethods reports whether the object declares any remote methods.
// Remote methods are recorded under "$remote$"-prefixed method keys.
func (o *Object) HasRemoteMethods() bool {
	for k := range o.methodKeys {
		if strings.HasPrefix(k, "$remote$") {
			return true
		}
	}
	return false
}
