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
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/runtime/internal/modules"
	"github.com/ballerina-nutcracker/ballerina/values"
)

func dereferenceAnnotationValue(ctx *extern.Context, value values.AnnotationValue) (values.AnnotationValue, bool) {
	ref, ok := value.(*values.RuntimeAnnotationValueRef)
	if !ok {
		return value, true
	}
	registry := ctx.Env.Registry.(*modules.Registry)
	module := registry.GetModuleByName(ref.Organization, ref.Module)
	if module == nil {
		return nil, false
	}
	value, ok = module.Globals[ref.GlobalLookupKey()]
	return value, ok
}

func resolveAnnotationValues(ctx *extern.Context, annotations values.AnnotationValues) (values.AnnotationValues, bool) {
	resolved := values.NewAnnotationValues()
	for key, value := range annotations {
		value, ok := dereferenceAnnotationValue(ctx, value)
		if !ok {
			return values.NewAnnotationValues(), false
		}
		resolved[key] = value
	}
	return resolved, true
}

// RecordFieldAnnotations resolves the runtime-visible annotation values attached
// to the field named field of the record type td denotes. The second return is
// false if the field carries no annotations or a runtime annotation value could
// not be loaded.
func RecordFieldAnnotations(ctx *extern.Context, td *values.TypeDesc, field string) (values.AnnotationValues, bool) {
	annotations, ok := td.FieldAnnotations[field]
	if !ok {
		return values.NewAnnotationValues(), false
	}
	return resolveAnnotationValues(ctx, annotations)
}

// AnnotatedRecordFields returns the names of the record fields of td that carry
// at least one runtime-visible annotation, in a deterministic order.
func AnnotatedRecordFields(_ *extern.Context, td *values.TypeDesc) []string {
	return td.FieldAnnotations.SortedFields()
}
