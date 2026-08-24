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

package extern_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/bir"
	bircodec "github.com/ballerina-nutcracker/ballerina/bir/codec"
	"github.com/ballerina-nutcracker/ballerina/birgen"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/desugar"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/model/symbolpool"
	"github.com/ballerina-nutcracker/ballerina/nodebuilder"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/test_util/langlib"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
	"github.com/ballerina-nutcracker/ballerina/values"

	_ "github.com/ballerina-nutcracker/ballerina/lib/rt"
)

func TestExternValid(t *testing.T) {
	externs := []testharness.ExternRegistration{
		{
			Org: "$anon", Module: "1-v", FuncName: "foo",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return "$foo", nil
			},
		},
		{
			Org: "$anon", Module: "1-v", FuncName: "bar",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				return values.String(args[0], nil) + ", " + values.String(args[1], nil), nil
			},
		},
	}
	runExtern(t, fileCase("1-v"), testharness.NewTestPal(), externs)
}

func TestInvokeNilFunctionValue(t *testing.T) {
	externs := []testharness.ExternRegistration{{
		Org: "$anon", Module: "invoke-nil-function-v", FuncName: "invokeNilFunction",
		Impl: func(ctx *extern.Context, _ []values.BalValue) (values.BalValue, error) {
			_, err := ctx.InvokeFunctionValue(nil, nil)
			return err != nil, nil
		},
	}}
	runExtern(t, fileCase("invoke-nil-function-v"), testharness.NewTestPal(), externs)
}

func TestExternTypeMismatchArg(t *testing.T) {
	runExtern(t, fileCase("2-e"), testharness.NewTestPal(), nil)
}

func TestExternTypeMismatchReturn(t *testing.T) {
	runExtern(t, fileCase("3-e"), testharness.NewTestPal(), nil)
}

func TestDependentlyTyped(t *testing.T) {
	const org, mod = "$anon", "dependently-typed-v"
	externs := []testharness.ExternRegistration{
		{Org: org, Module: mod, FuncName: "inferred", Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			td, ok := args[1].(*values.TypeDesc)
			if !ok {
				return nil, fmt.Errorf("expected typedesc argument, got %T", args[1])
			}
			if got := values.String(td, nil); got != "typedesc" {
				return nil, fmt.Errorf("expected typedesc string, got %q", got)
			}
			if !semtypes.IsSubtype(ctx.TypeCtx(), values.SemTypeForValue(td), semtypes.Typedesc) {
				return nil, fmt.Errorf("expected typedesc semtype")
			}
			switch {
			case semtypes.IsSubtype(ctx.TypeCtx(), td.Type, semtypes.Int):
				return int64(1), nil
			case semtypes.IsSubtype(ctx.TypeCtx(), td.Type, semtypes.String):
				return "foo", nil
			}
			panic(values.NewErrorWithMessage("unsupported inferred typedesc constraint"))
		}},
		{Org: org, Module: mod, FuncName: "inferredSubType", Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			td, ok := args[1].(*values.TypeDesc)
			if !ok {
				return nil, fmt.Errorf("expected typedesc argument, got %T", args[1])
			}
			if !semtypes.IsSubtype(ctx.TypeCtx(), td.Type, semtypes.Int) {
				panic(values.NewErrorWithMessage("inferredSubType requires typedesc<int>"))
			}
			return int64(1), nil
		}},
		{Org: org, Module: mod, FuncName: "inferredPartially", Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			td, ok := args[1].(*values.TypeDesc)
			if !ok {
				return nil, fmt.Errorf("expected typedesc argument, got %T", args[1])
			}
			switch {
			case semtypes.IsSubtype(ctx.TypeCtx(), semtypes.Int, td.Type):
				return int64(0), nil
			case semtypes.IsSubtype(ctx.TypeCtx(), semtypes.String, td.Type):
				return "bar", nil
			}
			panic(values.NewErrorWithMessage("unsupported inferredPartially typedesc constraint"))
		}},
		{Org: org, Module: mod, FuncName: "shiftBy", Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			src, ok := args[0].(*values.Map)
			if !ok {
				return nil, fmt.Errorf("expected record argument, got %T", args[0])
			}
			dx := args[1].(int64)
			dy := args[2].(int64)
			td, ok := args[3].(*values.TypeDesc)
			if !ok {
				return nil, fmt.Errorf("expected typedesc argument, got %T", args[3])
			}
			xVal, _ := src.Get("x")
			yVal, _ := src.Get("y")
			atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx(), td.Type)
			return values.NewMap(td.Type, atomic, false, []values.MapEntry{
				{Key: "x", Value: xVal.(int64) + dx},
				{Key: "y", Value: yVal.(int64) + dy},
			}), nil
		}},
		{Org: org, Module: mod, FuncName: "inferredWithDefault", Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			val, ok := args[0].(int64)
			if !ok {
				return nil, fmt.Errorf("expected int argument, got %T", args[0])
			}
			td, ok := args[1].(*values.TypeDesc)
			if !ok {
				return nil, fmt.Errorf("expected typedesc argument, got %T", args[1])
			}
			switch {
			case semtypes.IsSubtype(ctx.TypeCtx(), td.Type, semtypes.Int):
				return val, nil
			case semtypes.IsSubtype(ctx.TypeCtx(), td.Type, semtypes.String):
				return fmt.Sprintf("%d", val), nil
			}
			panic(values.NewErrorWithMessage("unsupported inferredWithDefault typedesc constraint"))
		}},
		{Org: org, Module: mod, FuncName: "inferredMaybeError", Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			td, ok := args[0].(*values.TypeDesc)
			if !ok {
				return nil, fmt.Errorf("expected typedesc argument, got %T", args[0])
			}
			// return an error to verify the inferred typedesc was widened to include error
			if semtypes.IsSubtype(ctx.TypeCtx(), semtypes.Error, td.Type) {
				return values.NewErrorWithMessage("error"), nil
			}
			panic(values.NewErrorWithMessage("inferredMaybeError: expected error to be in typedesc"))
		}},
		{Org: org, Module: mod, FuncName: "Getter." + model.RemoteMethodName("get"), Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			td, ok := args[1].(*values.TypeDesc)
			if !ok {
				return nil, fmt.Errorf("expected typedesc argument, got %T", args[1])
			}
			if !semtypes.IsSubtype(ctx.TypeCtx(), semtypes.String, td.Type) {
				panic(values.NewErrorWithMessage("Getter.get: expected string-compatible typedesc"))
			}
			return "immutable", nil
		}},
	}
	runExtern(t, fileCase("dependently-typed-v"), testharness.NewTestPal(), externs)
}

func TestDependentlyTypedAlias(t *testing.T) {
	aliasImpl := func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		if _, ok := args[1].(*values.TypeDesc); !ok {
			return nil, fmt.Errorf("expected typedesc argument, got %T", args[1])
		}
		switch args[0].(int64) {
		case 0, 2:
			return int64(10), nil
		case 1, 3:
			return "alias", nil
		}
		panic(values.NewErrorWithMessage("unsupported alias typedesc constraint"))
	}
	const org, mod = "$anon", "dependent-alias-v"
	externs := []testharness.ExternRegistration{
		{Org: org, Module: mod, FuncName: "viaAlias", Impl: aliasImpl},
		{Org: org, Module: mod, FuncName: "viaAliasUnion", Impl: aliasImpl},
		{Org: org, Module: mod, FuncName: "viaChainedAlias", Impl: aliasImpl},
	}
	runExtern(t, fileCase("dependent-alias-v"), testharness.NewTestPal(), externs)
}

func TestDependentlyTypedIncludedRecordParam(t *testing.T) {
	externs := []testharness.ExternRegistration{{
		Org: "$anon", Module: "dependently-typed-incl-record-v", FuncName: "shift",
		Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			src, ok := args[0].(*values.Map)
			if !ok {
				return nil, fmt.Errorf("expected record argument, got %T", args[0])
			}
			opts, ok := args[1].(*values.Map)
			if !ok {
				return nil, fmt.Errorf("expected record argument, got %T", args[1])
			}
			td, ok := args[2].(*values.TypeDesc)
			if !ok {
				return nil, fmt.Errorf("expected typedesc argument, got %T", args[2])
			}
			xVal, _ := src.Get("x")
			yVal, _ := src.Get("y")
			dxVal, _ := opts.Get("dx")
			dyVal, _ := opts.Get("dy")
			out := values.NewMap(td.Type, semtypes.ToMappingAtomicType(ctx.TypeCtx(), td.Type), false, nil)
			out.Put(ctx.TypeCtx(), "x", xVal.(int64)+dxVal.(int64))
			out.Put(ctx.TypeCtx(), "y", yVal.(int64)+dyVal.(int64))
			return out, nil
		},
	}}
	runExtern(t, fileCase("dependently-typed-incl-record-v"), testharness.NewTestPal(), externs)
}

func TestDependentlyTypedMethod(t *testing.T) {
	externs := []testharness.ExternRegistration{{
		Org: "testorg", Module: "crossmoduledependentfn.http",
		FuncName: "Client." + model.RemoteMethodName("get"),
		Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			td, ok := args[3].(*values.TypeDesc)
			if !ok {
				return nil, fmt.Errorf("expected typedesc argument, got %T", args[3])
			}
			switch {
			case semtypes.IsSubtype(ctx.TypeCtx(), semtypes.String, td.Type):
				return "string response", nil
			case semtypes.IsSubtype(ctx.TypeCtx(), semtypes.Int, td.Type):
				return int64(2), nil
			}
			panic(values.NewErrorWithMessage("unsupported targetType"))
		},
	}}
	runExtern(t, projectCase("dependently-typed-method-v"), testharness.NewTestPal(), externs)
}

func TestExternResourceMethod(t *testing.T) {
	externs := []testharness.ExternRegistration{
		{
			Org: "testorg", Module: "externresourcemethod.api", FuncName: "Client.$resource$get$0",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				return "items/" + values.String(args[1], nil), nil
			},
		},
		{
			Org: "testorg", Module: "externresourcemethod.api", FuncName: "Client.$resource$get$1",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				return args[2].(int64) * 2, nil
			},
		},
	}
	runExtern(t, projectCase("resource-method-v"), testharness.NewTestPal(), externs)
}

func TestListenerDispatch(t *testing.T) {
	// `trigger` writes through pal.IO.Stdout directly to mirror what
	// io:println does at runtime, without requiring a closure over the
	// *runtime.Runtime (which is built inside testharness.Run).
	externs := []testharness.ExternRegistration{
		{
			Org: "testorg", Module: "externlistener.lst", FuncName: "Listener.attach",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				args[0].(*values.Object).Put("svc", args[1].(*values.Object))
				return nil, nil
			},
		},
		{
			Org: "testorg", Module: "externlistener.lst", FuncName: "Listener.trigger",
			Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
				receiver := args[0].(*values.Object)
				svcVal, ok := receiver.Get("svc")
				if !ok {
					return nil, fmt.Errorf("listener has no attached service")
				}
				svc := svcVal.(*values.Object)

				rh, ok := ctx.LookupResourceMethod(svc, "get", []values.BalValue{"greeting", "world"})
				if !ok {
					return nil, fmt.Errorf("resource method 'get greeting/[name]' not found")
				}
				out, err := ctx.InvokeMethod(rh, nil)
				if err != nil {
					return nil, err
				}
				_, _ = ctx.Env.Platform.IO.Stdout([]byte(values.String(out, nil) + "\n"))

				mh, ok := ctx.LookupRemoteMethod(svc, "shutdown")
				if !ok {
					return nil, fmt.Errorf("remote method 'shutdown' not found")
				}
				out, err = ctx.InvokeMethod(mh, []values.BalValue{svc})
				if err != nil {
					return nil, err
				}
				_, _ = ctx.Env.Platform.IO.Stdout([]byte(values.String(out, nil) + "\n"))
				return nil, nil
			},
		},
	}
	runExtern(t, projectCase("listener-dispatch-v"), testharness.NewTestPal(), externs)
}

func TestAnnotationRuntimeMetadata(t *testing.T) {
	const org, module = "testorg", "annotationruntime"
	serviceKey := model.AnnotationKey(model.PackageIdentifier{
		Organization: org,
		Package:      module + ".meta",
		Version:      "0.1.0",
	}, "serviceMeta")
	parameterKey := model.AnnotationKey(model.PackageIdentifier{
		Organization: org,
		Package:      module + ".meta",
		Version:      "0.1.0",
	}, "parameterMeta")
	markerKey := model.AnnotationKey(model.PackageIdentifier{
		Organization: org,
		Package:      module + ".meta",
		Version:      "0.1.0",
	}, "marker")

	attachedService := func(receiver *values.Object) (*values.Object, error) {
		svc, ok := receiver.Get("svc")
		if !ok {
			return nil, fmt.Errorf("listener has no attached service")
		}
		return svc.(*values.Object), nil
	}
	annotationName := func(value values.AnnotationValue) (string, error) {
		mapping, ok := value.(*values.Map)
		if !ok {
			return "", fmt.Errorf("annotation value has type %T", value)
		}
		name, ok := mapping.Get("name")
		if !ok {
			return "", fmt.Errorf("annotation value has no name")
		}
		return name.(string), nil
	}

	externs := []testharness.ExternRegistration{
		{Org: org, Module: module + ".lst", FuncName: "Listener.inspect",
			Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
				listener := args[0].(*values.Object)
				methodHandle, ok := ctx.LookupObjectMethod(listener, "inspect")
				if !ok {
					return nil, fmt.Errorf("method 'inspect' not found")
				}
				methodSignature, ok := ctx.MethodSignature(methodHandle)
				if !ok || len(methodSignature.Params) != 0 {
					return nil, fmt.Errorf("unexpected native method signature: %#v", methodSignature)
				}
				methodMetadata, ok := ctx.MethodMetadata(methodHandle)
				if !ok || len(methodMetadata.Params) != 0 {
					return nil, fmt.Errorf("unexpected native method metadata: %#v", methodMetadata)
				}

				functionHandle, ok := ctx.LookupFunction(org, module, "parameterName")
				if !ok {
					return nil, fmt.Errorf("function 'parameterName' not found")
				}
				functionSignature, ok := ctx.FunctionSignature(functionHandle)
				if !ok || len(functionSignature.Params) != 0 ||
					!semtypes.IsSubtype(ctx.TypeCtx(), functionSignature.ReturnType, semtypes.String) {
					return nil, fmt.Errorf("unexpected function signature: %#v", functionSignature)
				}
				functionMetadata, ok := ctx.FunctionMetadata(functionHandle)
				if !ok || len(functionMetadata.Params) != 0 {
					return nil, fmt.Errorf("unexpected function metadata: %#v", functionMetadata)
				}

				svc, err := attachedService(listener)
				if err != nil {
					return nil, err
				}
				annotations, ok := ctx.ObjectAnnotations(svc)
				if !ok {
					return nil, fmt.Errorf("service annotations are unavailable")
				}
				serviceName, err := annotationName(annotations[serviceKey])
				if err != nil {
					return nil, err
				}

				handle, ok := ctx.LookupResourceMethod(svc, "get", []values.BalValue{"items"})
				if !ok {
					return nil, fmt.Errorf("resource method 'get items' not found")
				}
				signature, ok := ctx.MethodSignature(handle)
				if !ok || len(signature.Params) != 2 || signature.RestParam == nil {
					return nil, fmt.Errorf("unexpected resource signature: %#v", signature)
				}
				metadata, ok := ctx.MethodMetadata(handle)
				if !ok || len(metadata.Params) != 2 || metadata.RestParam == nil {
					return nil, fmt.Errorf("unexpected resource metadata: %#v", metadata)
				}
				countName, err := annotationName(metadata.Params[0].Annotations[parameterKey])
				if err != nil {
					return nil, err
				}
				headerName, err := annotationName(metadata.Params[1].Annotations[parameterKey])
				if err != nil {
					return nil, err
				}
				if signature.Params[0].Name != "count" || !semtypes.IsSubtype(ctx.TypeCtx(), signature.Params[0].Type, semtypes.Int) {
					return nil, fmt.Errorf("unexpected count descriptor")
				}
				if signature.Params[1].Name != "header" || !semtypes.IsSubtype(ctx.TypeCtx(), signature.Params[1].Type, semtypes.String) {
					return nil, fmt.Errorf("unexpected header descriptor")
				}
				if !semtypes.IsSubtype(ctx.TypeCtx(), signature.ReturnType, semtypes.Int) {
					return nil, fmt.Errorf("unexpected return type")
				}
				if signature.RestParam.Name != "extras" || metadata.RestParam.Annotations[markerKey] != true {
					return nil, fmt.Errorf("unexpected rest descriptor")
				}

				for _, line := range []string{serviceName, countName, headerName, "rest-marker"} {
					_, _ = ctx.Env.Platform.IO.Stdout([]byte(line + "\n"))
				}
				return nil, nil
			}},
		{Org: org, Module: module + ".lst", FuncName: "Listener.invokeWithoutArgs",
			Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
				svc, err := attachedService(args[0].(*values.Object))
				if err != nil {
					return nil, err
				}
				handle, ok := ctx.LookupResourceMethod(svc, "get", []values.BalValue{"items"})
				if !ok {
					return nil, fmt.Errorf("resource method 'get items' not found")
				}
				return ctx.InvokeMethod(handle, nil)
			}},
	}
	runExtern(t, projectCase("annotation-runtime-v"), testharness.NewTestPal(), externs)
}

func TestStartMethod(t *testing.T) {
	externs := []testharness.ExternRegistration{
		{
			Org: "testorg", Module: "startmethod.lst", FuncName: "Listener.attach",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				args[0].(*values.Object).Put("svc", args[1].(*values.Object))
				return nil, nil
			},
		},
		{
			Org: "testorg", Module: "startmethod.lst", FuncName: "Listener.trigger",
			Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
				receiver := args[0].(*values.Object)
				svcVal, ok := receiver.Get("svc")
				if !ok {
					return nil, fmt.Errorf("listener has no attached service")
				}
				svc := svcVal.(*values.Object)

				rh, ok := ctx.LookupResourceMethod(svc, "get", []values.BalValue{"greeting", "world"})
				if !ok {
					return nil, fmt.Errorf("resource method 'get greeting/[name]' not found")
				}
				resCh, err := ctx.StartMethod(rh, nil)
				if err != nil {
					return nil, err
				}

				mh, ok := ctx.LookupRemoteMethod(svc, "shutdown")
				if !ok {
					return nil, fmt.Errorf("remote method 'shutdown' not found")
				}
				remCh, err := ctx.StartMethod(mh, []values.BalValue{svc})
				if err != nil {
					return nil, err
				}

				_, _ = ctx.Env.Platform.IO.Stdout([]byte(values.String(<-resCh, nil) + "\n"))
				_, _ = ctx.Env.Platform.IO.Stdout([]byte(values.String(<-remCh, nil) + "\n"))
				return nil, nil
			},
		},
	}
	runExtern(t, projectCase("start-method-v"), testharness.NewTestPal(), externs)
}

func TestStartMethodError(t *testing.T) {
	externs := []testharness.ExternRegistration{
		{
			Org: "testorg", Module: "startmethoderror.lst", FuncName: "Listener.attach",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				args[0].(*values.Object).Put("svc", args[1].(*values.Object))
				return nil, nil
			},
		},
		{
			Org: "testorg", Module: "startmethoderror", FuncName: "$service$0." + model.RemoteMethodName("boom"),
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return nil, fmt.Errorf("boom")
			},
		},
		{
			Org: "testorg", Module: "startmethoderror.lst", FuncName: "Listener.trigger",
			Impl: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
				receiver := args[0].(*values.Object)
				svcVal, ok := receiver.Get("svc")
				if !ok {
					return nil, fmt.Errorf("listener has no attached service")
				}
				svc := svcVal.(*values.Object)

				boomH, ok := ctx.LookupRemoteMethod(svc, "boom")
				if !ok {
					return nil, fmt.Errorf("remote method 'boom' not found")
				}
				boomCh, err := ctx.StartMethod(boomH, []values.BalValue{svc})
				if err != nil {
					return nil, err
				}
				boomVal := <-boomCh
				boomErr, ok := boomVal.(*values.Error)
				if !ok {
					return nil, fmt.Errorf("expected *values.Error, got %T", boomVal)
				}
				if !strings.Contains(boomErr.Message, "boom") {
					return nil, fmt.Errorf("expected message to contain 'boom', got %q", boomErr.Message)
				}
				_, _ = ctx.Env.Platform.IO.Stdout([]byte("got error: " + boomErr.Message + "\n"))

				// A follow-up StartMethod after an error-returning strand still works.
				okH, ok := ctx.LookupRemoteMethod(svc, "ok")
				if !ok {
					return nil, fmt.Errorf("remote method 'ok' not found")
				}
				okCh, err := ctx.StartMethod(okH, []values.BalValue{svc})
				if err != nil {
					return nil, err
				}
				_, _ = ctx.Env.Platform.IO.Stdout([]byte(values.String(<-okCh, nil) + "\n"))
				return nil, nil
			},
		},
	}
	runExtern(t, projectCase("start-method-error-v"), testharness.NewTestPal(), externs)
}

func TestExternHandle(t *testing.T) {
	type myHandle struct {
		data string
	}
	externs := []testharness.ExternRegistration{
		{
			Org: "$anon", Module: "4-v", FuncName: "createHandle",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return &myHandle{data: "handle_value"}, nil
			},
		},
		{
			Org: "$anon", Module: "4-v", FuncName: "useHandle",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				return args[0].(*myHandle).data, nil
			},
		},
	}
	runExtern(t, fileCase("4-v"), testharness.NewTestPal(), externs)
}

func getBallerinaEnvPath(t *testing.T) string {
	if balEnv := os.Getenv(projects.BallerinaEnvVar); balEnv != "" {
		return balEnv
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get user home: %v", err)
	}
	return filepath.Join(userHome, projects.UserHomeDirName)
}

// TestDependentlyTypedCrossModuleRoundtrip loads a project containing a
// dependently-typed extern in a helper module, serializes dependency exported
// symbols and BIR, then recompiles the main module against the deserialized
// symbols. This validates that dependent-return and inferred-typedesc-default
// metadata survive serialization across a dependency boundary.
func TestDependentlyTypedCrossModuleRoundtrip(t *testing.T) {
	projectDir := filepath.Join(testDataDir, "cross-module-dependent-fn-v")
	mainBalPath := filepath.Join(projectDir, "main.bal")
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatal(err)
	}

	const (
		org         = "testorg"
		packageRoot = "crossmoduledependentfn"
		helperMod   = "crossmoduledependentfn.helper"
	)

	ballerinaEnvFs := os.DirFS(getBallerinaEnvPath(t))
	result, err := projects.Load(os.DirFS(absProjectDir), ".", projects.ProjectLoadConfig{
		BallerinaEnvFs: ballerinaEnvFs,
	})
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}
	if result.Diagnostics().HasErrors() {
		for _, d := range result.Diagnostics().Diagnostics() {
			t.Logf("project diagnostic: %v", d)
		}
		t.Fatal("project load had errors")
	}

	project := result.Project()
	currentPkg := project.CurrentPackage()
	compilation := currentPkg.Compilation()
	if compilation.DiagnosticResult().HasErrors() {
		for _, d := range compilation.DiagnosticResult().Diagnostics() {
			t.Logf("diagnostic: %v", d)
		}
		t.Fatal("compilation had errors")
	}

	backend := projects.NewBallerinaBackend(compilation)
	birPkgs := backend.BIRPackages()
	if len(birPkgs) == 0 {
		t.Fatal("compilation succeeded but produced no BIR packages")
	}

	freshEnv := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	publicSymbols := make(map[semantics.PackageIdentifier]model.ExportedSymbolSpace)
	deserializedPkgs := make([]*bir.BIRPackage, 0, len(birPkgs))
	mainPkg := backend.BIR()
	exportedSymbols := backend.ExportedSymbols()
	typeEnv := project.Environment().TypeEnv()

	for _, pkg := range birPkgs {
		if pkg == mainPkg {
			continue
		}

		pkgIdent := semantics.PackageIdentifier{
			OrgName:    pkg.PackageID.OrgName.Value(),
			ModuleName: pkg.PackageID.PkgName.Value(),
		}
		exported, ok := exportedSymbols[pkgIdent]
		if !ok {
			t.Fatalf("exported symbols not found for %s/%s", pkgIdent.OrgName, pkgIdent.ModuleName)
		}

		symBytes, err := symbolpool.Marshal(exported, project.Environment().CompilerEnvironment())
		if err != nil {
			t.Fatalf("symbol Marshal for %s/%s: %v", pkgIdent.OrgName, pkgIdent.ModuleName, err)
		}
		deserializedExported, err := symbolpool.Unmarshal(freshEnv, symBytes)
		if err != nil {
			t.Fatalf("symbol Unmarshal for %s/%s: %v", pkgIdent.OrgName, pkgIdent.ModuleName, err)
		}
		publicSymbols[pkgIdent] = deserializedExported

		birBytes, err := bircodec.Marshal(typeEnv, pkg)
		if err != nil {
			t.Fatalf("BIR Marshal for %s/%s: %v", pkgIdent.OrgName, pkgIdent.ModuleName, err)
		}
		deserializedPkg, err := bircodec.Unmarshal(context.NewCompilerContext(freshEnv), birBytes)
		if err != nil {
			t.Fatalf("BIR Unmarshal for %s/%s: %v", pkgIdent.OrgName, pkgIdent.ModuleName, err)
		}
		deserializedPkgs = append(deserializedPkgs, deserializedPkg)
	}

	_, mainBIR := compileSingleFileModule(t, freshEnv, mainBalPath,
		model.Name(org),
		[]model.Name{model.Name(packageRoot)},
		publicSymbols,
		org,
	)
	deserializedPkgs = append(deserializedPkgs, mainBIR)

	// Stage 5: interpret [deserialized dependency BIR, freshly compiled main BIR]
	// with helper's native registered. Validate stdout.
	pal := testharness.NewTestPal()
	defer pal.Close()
	rt := runtime.NewRuntime(pal.Platform(), freshEnv.GetTypeEnv())
	tyCtx := semtypes.ContextFrom(rt.GetTypeEnv())
	runtime.RegisterExternFunction(rt, org, helperMod, "inferred", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		td, ok := args[1].(*values.TypeDesc)
		if !ok {
			return nil, fmt.Errorf("expected typedesc argument, got %T", args[1])
		}
		switch {
		case semtypes.IsSubtype(tyCtx, td.Type, semtypes.Int):
			return int64(1), nil
		case semtypes.IsSubtype(tyCtx, td.Type, semtypes.String):
			return "foo", nil
		}
		panic(values.NewErrorWithMessage("unsupported inferred typedesc constraint"))
	})
	for _, pkg := range deserializedPkgs {
		if err := rt.Init(*pkg); err != nil {
			t.Fatalf("runtime error: %v", err)
		}
	}
	rt.Listen()
	<-rt.ExitStatus

	const expected = "1\nfoo\n"
	if got := pal.Stdout(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// compileSingleFileModule parses a .bal file and runs the full compilation
// pipeline for it as its own module with the given package identity. The
// publicSymbols map is consulted for any imports. Returns the module's
// exported symbol space and BIR package.
func compileSingleFileModule(
	t *testing.T,
	env *context.CompilerEnvironment,
	balPath string,
	orgName model.Name,
	nameComps []model.Name,
	publicSymbols map[semantics.PackageIdentifier]model.ExportedSymbolSpace,
	defaultOrg string,
) (model.ExportedSymbolSpace, *bir.BIRPackage) {
	t.Helper()
	absPath, err := filepath.Abs(balPath)
	if err != nil {
		t.Fatal(err)
	}
	cx := context.NewCompilerContext(env)
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading %s: %v", balPath, err)
	}
	cx.DiagnosticEnv().RegisterFile(absPath, text.NewStringTextDocument(string(content)))
	st, err := parser.GetSyntaxTree(cx, absPath, string(content))
	if err != nil {
		t.Fatalf("parsing %s: %v", balPath, err)
	}
	assertNoDiagnostics(t, cx, "Parse")
	cu := nodebuilder.GetCompilationUnit(cx, st)
	pkgID := cx.NewPackageID(orgName, nameComps, model.DEFAULT_VERSION)
	cu.SetPackageID(pkgID)
	compilationUnits := []*ast.BLangCompilationUnit{cu}

	langlibs, err := langlib.Build(cx, publicSymbols)
	if err != nil {
		t.Fatalf("loading lang libraries failed: %v", err)
	}
	pkgScope, exported, importedSymbols := semantics.ResolveSymbols(
		cx,
		*pkgID,
		compilationUnits,
		langlibs.ImplicitImports,
		langlibs.PublicSymbols,
		defaultOrg,
	)
	assertNoDiagnostics(t, cx, "ResolveSymbols")
	pkg := nodebuilder.ToPackageFromCompilationUnits(cx, compilationUnits)
	assertNoDiagnostics(t, cx, "ToPackageFromCompilationUnits")
	pkg.PackageID = pkgID
	pkg.Scope = pkgScope
	pkg.Imports = nil
	semantics.ResolvePublicNodeTypes(cx, pkg, importedSymbols)
	assertNoDiagnostics(t, cx, "ResolvePublicNodeTypes")
	semantics.ResolvePrivateNodesTypes(cx, pkg, importedSymbols)
	assertNoDiagnostics(t, cx, "ResolvePrivateNodesTypes")
	semantics.AnalyzeSemantics(cx, pkg, importedSymbols)
	assertNoDiagnostics(t, cx, "AnalyzeSemantics")
	cfg := semantics.CreateControlFlowGraph(cx, pkg)
	assertNoDiagnostics(t, cx, "CreateControlFlowGraph")
	semantics.AnalyzeCFG(cx, pkg, cfg)
	assertNoDiagnostics(t, cx, "AnalyzeCFG")
	pkg = desugar.DesugarPackage(cx, pkg, importedSymbols)
	assertNoDiagnostics(t, cx, "DesugarPackage")
	birPkg := birgen.GenBir(cx, pkg)
	if birPkg == nil {
		assertNoDiagnostics(t, cx, "GenBir")
		t.Fatal("BIR generation failed without a diagnostic")
	}
	return exported, birPkg
}

func assertNoDiagnostics(t *testing.T, cx *context.CompilerContext, stage string) {
	t.Helper()
	if !cx.HasDiagnostics() {
		return
	}
	for _, d := range cx.Diagnostics() {
		t.Logf("%s diagnostic: %s", stage, d)
	}
	t.Fatalf("%s produced diagnostics", stage)
}
