// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package signature

import "github.com/ballerina-nutcracker/ballerina/ast"

func nodeChainsAtOffset(pkg *ast.BLangPackage, offset, fileIndex int) [][]ast.BLangNode {
	finder := &chainFinder{offset: offset, fileIndex: fileIndex}
	ast.Walk(finder, pkg)
	return finder.candidates
}

type chainFinder struct {
	offset     int
	fileIndex  int
	stack      []ast.BLangNode
	candidates [][]ast.BLangNode
}

func (f *chainFinder) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		if len(f.stack) > 0 {
			f.stack = f.stack[:len(f.stack)-1]
		}
		return f
	}
	loc := node.GetPosition()
	if locationHasUsableOffsets(loc) && (!locationContains(loc, f.offset) || loc.FileIndex() != f.fileIndex) {
		return nil
	}
	f.stack = append(f.stack, node)
	if locationHasUsableOffsets(loc) {
		f.retainPath()
	}
	return f
}

func (f *chainFinder) retainPath() {
	loc := f.stack[len(f.stack)-1].GetPosition()
	for _, candidate := range f.candidates {
		other := candidate[len(candidate)-1].GetPosition()
		if rangeWithin(other, loc) && (!rangeWithin(loc, other) || pathPrefix(f.stack, candidate)) {
			return
		}
	}
	kept := f.candidates[:0]
	for _, candidate := range f.candidates {
		other := candidate[len(candidate)-1].GetPosition()
		if rangeWithin(loc, other) && (!rangeWithin(other, loc) || pathPrefix(candidate, f.stack)) {
			continue
		}
		kept = append(kept, candidate)
	}
	f.candidates = append(kept, append([]ast.BLangNode(nil), f.stack...))
}

func rangeWithin(inner, outer ast.Location) bool {
	return inner.StartOffset() >= outer.StartOffset() && inner.EndOffset() <= outer.EndOffset()
}

func pathPrefix(prefix, path []ast.BLangNode) bool {
	if len(prefix) > len(path) {
		return false
	}
	for i, node := range prefix {
		if node != path[i] {
			return false
		}
	}
	return true
}

func (f *chainFinder) VisitTypeData(*ast.TypeData) ast.Visitor { return f }

func locationHasUsableOffsets(loc ast.Location) bool {
	return loc.StartOffset() >= 0 && loc.EndOffset() >= loc.StartOffset() && (loc.StartOffset() != 0 || loc.EndOffset() != 0)
}

func locationContains(loc ast.Location, offset int) bool {
	return loc.StartOffset() >= 0 && loc.EndOffset() >= 0 && loc.StartOffset() <= offset && offset <= loc.EndOffset()
}
