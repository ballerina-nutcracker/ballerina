// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"fmt"
	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/birgen"
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/desugar"
	"github.com/ballerina-nutcracker/ballerina/semantics"
)

type ResolvedModule struct{ *moduleBase }
type SemanticallyAnalyzedModule struct{ *moduleBase }
type ControlFlowModule struct {
	*moduleBase
	CFG *semantics.PackageCFG
}
type AnalyzedModule struct{ *moduleBase }
type DesugaredModule struct{ *moduleBase }

func ResolvePrivateNodes(ctx *Context, module *PartiallyResolvedModule) *ResolvedModule {
	if ctx.Err() != nil {
		return nil
	}
	cx := ctx.newCompilerContext(module.ID)
	cx.StartStage(compilercontext.StageLocalNodeResolution)
	semantics.ResolvePrivateNodesTypes(cx, module.PackageNode, module.imported)
	cx.EndStage()
	ctx.drainModule(cx, module.ID, compilercontext.StageLocalNodeResolution, false, 0)
	if cx.HasErrors() || ctx.Err() != nil {
		return nil
	}
	return &ResolvedModule{moduleBase: module.moduleBase}
}

func AnalyzeSemantics(ctx *Context, module *ResolvedModule) *SemanticallyAnalyzedModule {
	if ctx.Err() != nil {
		return nil
	}
	cx := ctx.newCompilerContext(module.ID)
	cx.StartStage(compilercontext.StageSemanticAnalysis)
	semantics.AnalyzeSemantics(cx, module.PackageNode, module.imported)
	cx.EndStage()
	ctx.drainModule(cx, module.ID, compilercontext.StageSemanticAnalysis, false, 0)
	if cx.HasErrors() || ctx.Err() != nil {
		return nil
	}
	return &SemanticallyAnalyzedModule{moduleBase: module.moduleBase}
}

func CreateControlFlowGraph(ctx *Context, module *SemanticallyAnalyzedModule) *ControlFlowModule {
	if ctx.Err() != nil {
		return nil
	}
	cx := ctx.newCompilerContext(module.ID)
	cx.StartStage(compilercontext.StageCFGCreation)
	cfg := semantics.CreateControlFlowGraph(cx, module.PackageNode)
	cx.EndStage()
	ctx.drainModule(cx, module.ID, compilercontext.StageCFGCreation, false, 0)
	if cx.HasErrors() || ctx.Err() != nil || cfg == nil {
		return nil
	}
	return &ControlFlowModule{moduleBase: module.moduleBase, CFG: cfg}
}

func AnalyzeControlFlowGraph(ctx *Context, module *ControlFlowModule) *AnalyzedModule {
	if ctx.Err() != nil {
		return nil
	}
	cx := ctx.newCompilerContext(module.ID)
	cx.StartStage(compilercontext.StageCFGAnalysis)
	semantics.AnalyzeCFG(cx, module.PackageNode, module.CFG)
	cx.EndStage()
	ctx.drainModule(cx, module.ID, compilercontext.StageCFGAnalysis, false, 0)
	if cx.HasErrors() || ctx.Err() != nil {
		return nil
	}
	return &AnalyzedModule{moduleBase: module.moduleBase}
}

func Desugar(ctx *Context, module *AnalyzedModule) *DesugaredModule {
	if ctx.Err() != nil {
		return nil
	}
	cx := ctx.newCompilerContext(module.ID)
	cx.StartStage(compilercontext.StageDesugaring)
	packageNode := desugar.DesugarPackage(cx, module.PackageNode, module.imported)
	cx.EndStage()
	ctx.drainModule(cx, module.ID, compilercontext.StageDesugaring, false, 0)
	if cx.HasErrors() || ctx.Err() != nil || packageNode == nil {
		return nil
	}
	base := module.moduleBase
	if packageNode != module.PackageNode {
		copied := *base
		copied.PackageNode = packageNode
		base = &copied
	}
	return &DesugaredModule{moduleBase: base}
}

func GenerateBIR(ctx *Context, module *DesugaredModule) *bir.BIRPackage {
	if ctx.Err() != nil {
		return nil
	}
	cx := ctx.newCompilerContext(module.ID)
	cx.StartStage(compilercontext.StageBIRGeneration)
	prefix := ""
	if !ctx.isRoot(module.ID.Package) {
		prefix = fmt.Sprintf("%s/%s/%s::", module.ID.Package.Org, module.ID.Name, module.ID.Package.Version)
	}
	result := birgen.GenBirWithSourcePrefix(cx, module.PackageNode, prefix)
	cx.EndStage()
	ctx.drainModule(cx, module.ID, compilercontext.StageBIRGeneration, false, 0)
	if cx.HasErrors() || ctx.Err() != nil {
		return nil
	}
	return result
}
