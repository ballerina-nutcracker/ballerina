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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

// fingerprint.go implements design item 4: a per-module public-API
// structural fingerprint, used to prune a reset module's dependents back out
// of the reset scope when its exported surface didn't actually change (e.g. a
// function-body-only edit). Built entirely from already-public compiler APIs
// — context.CompilerContext.GetTypeEnv/SymbolName/SymbolKind/SymbolType,
// semtypes.TypeCheckContext/ToString, and
// model.ExportedSymbolSpace.PublicMainSymbols/AnnotationSpaces — no
// compiler-package change needed.
package compile

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"

	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// fingerprint is a per-module public-API structural summary. valid is false
// when no reliable fingerprint could be computed — the module has
// unresolved-symbol errors (a degenerate type would produce a false-positive
// "changed" signal), or semtypes.ToString panicked on an unhandled type
// shape. Two invalid fingerprints are never considered equal to each other:
// an invalid fingerprint carries no information, so treating "unknown" as
// "unchanged" would incorrectly prune a dependent that may genuinely need
// resetting.
type fingerprint struct {
	value string
	valid bool
}

// equal reports whether a and b are both valid and carry the same value.
func (a fingerprint) equal(b fingerprint) bool {
	return a.valid && b.valid && a.value == b.value
}

// computeFingerprint builds ctx's exported public-API fingerprint: a sorted,
// deterministic hash of name+kind+ToString(type) for every exported symbol
// in exported. Returns an invalid fingerprint if ctx has errors or if
// rendering any symbol's type panics.
func computeFingerprint(ctx *context.CompilerContext, exported model.ExportedSymbolSpace) (fp fingerprint) {
	if ctx.HasErrors() {
		return fingerprint{}
	}
	defer func() {
		if recover() != nil {
			fp = fingerprint{}
		}
	}()

	tcx := semtypes.TypeCheckContext(ctx.GetTypeEnv())
	var entries []string

	for ref := range exported.PublicMainSymbols() {
		entries = append(entries, fingerprintEntry(ctx, tcx, ref))
	}
	for _, space := range exported.AnnotationSpaces {
		for ref := range space.Symbols() {
			if !space.SymbolAt(ref.Index).IsPublic() {
				continue
			}
			entries = append(entries, fingerprintEntry(ctx, tcx, ref))
		}
	}
	sort.Strings(entries)

	h := sha256.New()
	for _, e := range entries {
		h.Write([]byte(e))
		h.Write([]byte{0})
	}
	return fingerprint{value: hex.EncodeToString(h.Sum(nil)), valid: true}
}

// fingerprintEntry renders one exported symbol's name+kind+type as a single
// deterministic string.
func fingerprintEntry(ctx *context.CompilerContext, tcx semtypes.Context, ref model.SymbolRef) string {
	name := ctx.SymbolName(ref)
	kind := ctx.SymbolKind(ref)
	ty := ctx.SymbolType(ref)
	return name + "\x01" + strconv.FormatUint(uint64(kind), 10) + "\x01" + semtypes.ToString(tcx, ty)
}
