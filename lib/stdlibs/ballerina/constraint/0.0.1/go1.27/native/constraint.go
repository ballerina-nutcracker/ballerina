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

package native

import (
	"errors"
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "constraint"

	annotationPrefix = "ballerina/constraint:"
	rootPath         = "$"
)

func init() {
	runtime.RegisterModuleInitializer(initConstraintModule)
}

func initConstraintModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "validate", validateExtern(rt))
}

func validateExtern(rt *runtime.Runtime) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		target, _ := args[1].(*values.TypeDesc)
		value, convErr := convertToTarget(ctx, args[0], target.Type)
		if convErr != nil {
			return newModuleError(ctx, "newTypeConversionError", typeConversionErrorMessage, convErr)
		}
		// A partial result is still validated: one unloadable annotation must not hide the others.
		annotations, _ := ctx.TypeAnnotations(target)
		check := &validation{now: rt.Platform().Time.Now}
		if err := validateAnnotations(check, annotations, value); err != nil {
			var invalid *invalidConstraintError
			if errors.As(err, &invalid) {
				return newModuleError(ctx, "newError", invalid.message)
			}
			return newModuleError(ctx, "newError", unexpectedErrorMessage)
		}
		if len(check.failures) > 0 {
			return validationError(ctx, check.failures)
		}
		return value, nil
	}
}

// convertToTarget leaves a value that already fits the target untouched and otherwise clones it with the target type.
func convertToTarget(ctx *extern.Context, value values.BalValue, target semtypes.SemType) (values.BalValue, *values.Error) {
	tc := ctx.TypeCtx()
	if semtypes.IsSubtype(tc, values.SemTypeForValue(value), target) {
		return value, nil
	}
	return values.CloneWithType(tc, value, target)
}

// validateAnnotations applies the annotations on the type itself and then those on each record field. Annotations
// on the types of nested values are not visible at runtime, so they are not validated.
func validateAnnotations(check *validation, annotations extern.TypeAnnotations, value values.BalValue) error {
	if err := applyAll(check, annotations.Annotations, value, rootPath); err != nil {
		return err
	}
	record, ok := value.(*values.Map)
	if !ok {
		return nil
	}
	fields := make([]string, 0, len(annotations.Fields))
	for field := range annotations.Fields {
		fields = append(fields, field)
	}
	slices.Sort(fields)
	for _, field := range fields {
		// Absent optional fields and nil fields are not constrained.
		fieldValue, present := record.Get(field)
		if !present || fieldValue == nil {
			continue
		}
		if err := applyAll(check, annotations.Fields[field], fieldValue, rootPath+"."+field); err != nil {
			return err
		}
	}
	return nil
}

func applyAll(check *validation, annotations values.AnnotationValues, value values.BalValue, path string) error {
	keys := make([]string, 0, len(annotations))
	for key := range annotations {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if !strings.HasPrefix(key, annotationPrefix) {
			continue
		}
		tag := key[strings.LastIndex(key, ":")+1:]
		constraints, ok := annotations[key].(*values.Map)
		if !ok {
			continue
		}
		if err := check.apply(tag, constraints, value, path); err != nil {
			return err
		}
	}
	return nil
}
