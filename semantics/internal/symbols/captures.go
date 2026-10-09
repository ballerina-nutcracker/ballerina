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

package symbols

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
)

type captureRegionKind uint8

const (
	// captureRegionNone is a plain block. It neither captures nor accumulates.
	captureRegionNone captureRegionKind = iota
	// captureRegionClosure is a deferred body: a function nested in a function, a
	// lambda or a default expression. It captures the block-level symbols it refers
	// to but does not declare.
	captureRegionClosure
	// captureRegionAccum is a repeated region (while, foreach or query). It
	// accumulates the captures made by closures nested inside it.
	captureRegionAccum
)

// captureAnalyzer holds the capture group a block resolver records into. The
// group is non-zero exactly when the region kind is not captureRegionNone.
type captureAnalyzer struct {
	captureRegionKind captureRegionKind
	captureGroup      model.CaptureGroupRef
}

// newCaptureAnalyzer builds the analyzer for a region owned by owner. A node
// walked more than once keeps the group allocated on its first walk.
func newCaptureAnalyzer(ctx *context.CompilerContext, kind captureRegionKind, owner ast.CaptureGroupOwner) captureAnalyzer {
	if kind == captureRegionNone {
		return captureAnalyzer{captureRegionKind: kind, captureGroup: 0}
	}
	group := owner.CaptureGroup()
	if group == 0 {
		group = ctx.NewCaptureGroup()
		owner.SetCaptureGroup(group)
	}
	return captureAnalyzer{captureRegionKind: kind, captureGroup: group}
}

// recordCapture adds ref to the group of every region between this resolver and
// the resolver declaring ref.
func (bs *blockSymbolResolver) recordCapture(ref model.SymbolRef) {
	if bs.declares(ref) {
		return
	}
	if bs.captureGroup != 0 {
		bs.GetCtx().AddToCaptureGroup(bs.captureGroup, ref)
	}
	bs.parent.recordCapture(ref)
}

func (bs *blockSymbolResolver) declares(ref model.SymbolRef) bool {
	return ref.SpaceIndex == bs.scope.MainSpace().SpaceIndex()
}
