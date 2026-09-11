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
	value, ok = module.GetGlobal(ref.GlobalLookupKey())
	return value, ok
}

// resolveAnnotationValues dereferences the runtime-valued entries of
// annotations. An entry whose value cannot be loaded is left out and the second
// return is false; every other entry is still returned.
func resolveAnnotationValues(ctx *extern.Context, annotations values.AnnotationValues) (values.AnnotationValues, bool) {
	resolved := values.NewAnnotationValues()
	complete := true
	for key, value := range annotations {
		value, ok := dereferenceAnnotationValue(ctx, value)
		if !ok {
			complete = false
			continue
		}
		resolved[key] = value
	}
	return resolved, complete
}

// TypeAnnotations resolves the runtime-visible annotations of the type td
// denotes: those attached to the type itself and those attached to each of its
// record fields. The second return is false if a runtime annotation value could
// not be loaded; that value is left out, and everything that did load is still
// returned, so one broken field does not hide the others.
func TypeAnnotations(ctx *extern.Context, td *values.TypeDesc) (extern.TypeAnnotations, bool) {
	annotations, complete := resolveAnnotationValues(ctx, td.Annotations)
	fields := make(map[string]values.AnnotationValues, len(td.FieldAnnotations))
	for field, fieldAnnotations := range td.FieldAnnotations {
		resolved, ok := resolveAnnotationValues(ctx, fieldAnnotations)
		if !ok {
			complete = false
		}
		if len(resolved) > 0 {
			fields[field] = resolved
		}
	}
	return extern.TypeAnnotations{Annotations: annotations, Fields: fields}, complete
}
