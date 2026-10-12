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

package cfg

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// analyzeWorkerMessageTermination reports a return that lets a worker of a
// worker region terminate with success before it runs all its
// message actions: its peers would wait for an action that never happens. A
// return whose value is an error terminates the worker with failure, which
// its peers observe.
func analyzeWorkerMessageTermination(ctx *context.CompilerContext, pkg *ast.BLangPackage) {
	finder := &workerBodyFinder{}
	ast.Walk(finder, pkg)
	tyCtx := semtypes.ContextFrom(ctx.GetTypeEnv())
	for _, body := range finder.bodies {
		defaultStmts := append(append([]ast.StatementNode{}, body.InitStmts...), body.Region.DefaultWorker.Stmts...)
		checkWorkerMessageTermination(ctx, tyCtx, defaultStmts)
		checkRegionMessageTermination(ctx, tyCtx, body.Region)
	}
	for _, fork := range finder.forks {
		checkRegionMessageTermination(ctx, tyCtx, fork.Region)
	}
}

func checkRegionMessageTermination(ctx *context.CompilerContext, tyCtx semtypes.Context, region *ast.WorkerRegion) {
	for _, worker := range region.Workers {
		checkWorkerMessageTermination(ctx, tyCtx, worker.Body.Stmts)
	}
}

type workerBodyFinder struct {
	bodies []*ast.BLangBlockFunctionBody
	forks  []*ast.BLangFork
}

func (f *workerBodyFinder) Visit(node ast.BLangNode) ast.Visitor {
	switch node := node.(type) {
	case *ast.BLangBlockFunctionBody:
		if node.Region != nil {
			f.bodies = append(f.bodies, node)
		}
	case *ast.BLangFork:
		f.forks = append(f.forks, node)
	}
	return f
}

func (f *workerBodyFinder) VisitTypeData(_ *ast.TypeData) ast.Visitor { return f }

func checkWorkerMessageTermination(ctx *context.CompilerContext, tyCtx semtypes.Context, stmts []ast.StatementNode) {
	collector := &workerTerminationCollector{}
	for _, stmt := range stmts {
		ast.Walk(collector, stmt.(ast.BLangNode))
	}
	if len(collector.messageActions) == 0 {
		return
	}
	last := collector.messageActions[len(collector.messageActions)-1]
	for _, ret := range collector.returns {
		if returnsError(tyCtx, ret) {
			continue
		}
		retPos := ret.GetPosition()
		if retPos.EndOffset() <= last.StartOffset() {
			ctx.SemanticError("worker can terminate before running all its message actions", retPos)
		}
	}
}

func returnsError(tyCtx semtypes.Context, ret *ast.BLangReturn) bool {
	if ret.Expr == nil {
		return false
	}
	ty := ret.Expr.Type()
	return !semtypes.IsZero(ty) && semtypes.IsSubtype(tyCtx, ty, semtypes.Error)
}

// workerTerminationCollector collects, in source order, the message actions
// and returns of a worker's statements. Nested functions and nested workers
// belong to their own workers, so it doesn't descend into them.
type workerTerminationCollector struct {
	messageActions []diagnostics.Location
	returns        []*ast.BLangReturn
}

func (c *workerTerminationCollector) Visit(node ast.BLangNode) ast.Visitor {
	switch node := node.(type) {
	case *ast.BLangLambdaFunction, *ast.BLangFunction, *ast.BLangClassDefinition,
		*ast.BLangFork, *ast.BLangNamedWorkerDeclaration:
		return nil
	case *ast.BLangReturn:
		c.returns = append(c.returns, node)
	case *ast.BLangWorkerAsyncSendAction, *ast.BLangWorkerSyncSendAction, *ast.BLangWorkerReceiveAction,
		*ast.BLangWorkerMultipleReceiveAction, *ast.BLangWorkerFlushAction:
		c.messageActions = append(c.messageActions, node.GetPosition())
	}
	return c
}

func (c *workerTerminationCollector) VisitTypeData(_ *ast.TypeData) ast.Visitor { return c }
