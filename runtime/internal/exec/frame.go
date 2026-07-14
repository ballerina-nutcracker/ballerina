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
	"ballerina-lang-go/bir"
	"ballerina-lang-go/model"
	"ballerina-lang-go/runtime/extern"
	runtimeframe "ballerina-lang-go/runtime/internal/frame"
	"ballerina-lang-go/runtime/internal/modules"
	"ballerina-lang-go/values"
)

type Frame = runtimeframe.Frame

func resolveFrame(frame *Frame, address bir.Address) *Frame {
	if address.Mode == bir.AddressingModeAbsolute {
		f := frame
		for i := 0; i < address.BaseIndex; i++ {
			f = f.Parent()
		}
		return f
	}
	return frame
}

// Load retrieves the value at the given address in the frame.
func Load(frame *Frame, address bir.Address) values.BalValue {
	return resolveFrame(frame, address).Local(address.FrameIndex)
}

// Store sets the value at the given address in the frame.
func Store(frame *Frame, address bir.Address, value values.BalValue) {
	resolveFrame(frame, address).SetLocal(address.FrameIndex, value)
}

func getOperandValue(ctx *extern.Context, op *bir.BIROperand, currentFrame *Frame) values.BalValue {
	if gv, ok := op.VariableDcl.(*bir.BIRGlobalVariableDcl); ok {
		module := getModule(ctx, gv.PkgId)
		return module.Globals[gv.GlobalVarLookupKey]
	}
	return Load(currentFrame, op.Address)
}

func setOperandValue(ctx *extern.Context, op *bir.BIROperand, currentFrame *Frame, value values.BalValue) {
	if gv, ok := op.VariableDcl.(*bir.BIRGlobalVariableDcl); ok {
		module := getModule(ctx, gv.PkgId)
		module.Globals[gv.GlobalVarLookupKey] = value
	} else {
		Store(currentFrame, op.Address, value)
	}
}

func getModule(ctx *extern.Context, pkgId *model.PackageID) *modules.BIRModule {
	env := ctx.Env
	registry := env.Registry.(*modules.Registry)
	return registry.GetModule(pkgId)
}
