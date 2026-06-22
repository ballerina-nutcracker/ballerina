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

// AnnotationValue is the evaluated Ballerina value stored for an annotation.
type AnnotationValue = BalValue

// AnnotationValues maps fully-qualified annotation keys to evaluated values.
type AnnotationValues map[string]AnnotationValue

// RuntimeAnnotationValueRef identifies an annotation value initialized in a
// module global.
type RuntimeAnnotationValueRef struct {
	Organization string
	Module       string
	GlobalName   string
}

// GlobalLookupKey returns the runtime global key for the annotation value.
func (r *RuntimeAnnotationValueRef) GlobalLookupKey() string {
	return r.Organization + "/" + r.Module + ":" + r.GlobalName
}

// NewAnnotationValues returns an initialized annotation value map.
func NewAnnotationValues() AnnotationValues {
	return make(AnnotationValues)
}
