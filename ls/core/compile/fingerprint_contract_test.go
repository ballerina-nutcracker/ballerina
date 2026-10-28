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

package compile

import (
	"testing"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func TestFingerprintSupportedContractEdits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, before, after string }{
		{"export name", "public function f(int a) returns int { return a; }", "public function g(int a) returns int { return a; }"},
		{"parameter name", "public function f(int a) returns int { return a; }", "public function f(int b) returns int { return b; }"},
		{"parameter default", "public function f(int a = 1) returns int { return a; }", "public function f(int a) returns int { return a; }"},
		{"parameter type", "public function f(int a) returns int { _ = a; return 1; }", "public function f(string a) returns int { _ = a; return 1; }"},
		{"parameter order", "public function f(int a, int b) returns int { _ = b; return a; }", "public function f(int b, int a) returns int { _ = b; return a; }"},
		{"parameter count", "public function f(int a) returns int { return a; }", "public function f(int a, int b) returns int { _ = b; return a; }"},
		{"rest status", "public function f(int a) returns int { _ = a; return 1; }", "public function f(int... a) returns int { _ = a; return 1; }"},
		{"rest name", "public function f(int... a) returns int { _ = a; return 1; }", "public function f(int... b) returns int { _ = b; return 1; }"},
		{"rest type", "public function f(int... a) returns int { _ = a; return 1; }", "public function f(string... a) returns int { _ = a; return 1; }"},
		{"isolated flag", "public function f() returns int { return 1; }", "public isolated function f() returns int { return 1; }"},
		{"transactional flag", "public function f() returns int { return 1; }", "public transactional function f() returns int { return 1; }"},
		{"return type", "public function f() returns int { return 1; }", "public function f() returns string { return \"a\"; }"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a, b := buildFingerprint(t, tc.before), buildFingerprint(t, tc.after)
			if !a.valid || !b.valid {
				t.Fatalf("supported contracts must be valid: %+v, %+v", a, b)
			}
			if a.equal(b) {
				t.Fatal("contract edit must change fingerprint")
			}
		})
	}
}

func TestFingerprintUnsupportedContracts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, source string }{
		{"record", "public type R record {| int a = 1; |};"},
		{"object", "public type O object { int a; };"},
		{"constant", "public const N = 1;"},
		{"variable", "public int n = 1;"},
		{"annotation", "public annotation A on function;"},
		{"alias", "type I int; public function f(I a) returns int { _ = a; return 1; }"},
		{"named return", "type I int; public function f() returns I { return 1; }"},
		{"composite", "public function f(int[] a) returns int { _ = a; return 1; }"},
		{"recursive", "public type R record {| R? next; |};"},
		{"subtype", "public function f(byte a) returns int { return a; }"},
		{"unsupported builtin", "public function f(boolean a) returns int { _ = a; return 1; }"},
		{"nil return", "public function f() {}"},
		{"attachment", "annotation A on function; @A public function f() returns int { return 1; }"},
		{"parameter attachment", "annotation A on parameter; public function f(@A int a) returns int { return a; }"},
		{"return attachment", "annotation A on return; public function f() returns @A int { return 1; }"},
		{"included record", "type R record {| int a; |}; public function f(*R r) returns int { _ = r; return 1; }"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fp := buildFingerprint(t, tc.source)
			if fp.valid || fp.equal(fp) {
				t.Fatalf("unsupported contracts must fail closed: %+v", fp)
			}
			supported := buildFingerprint(t, "public function f() returns int { return 1; }")
			if supported.equal(fp) || fp.equal(supported) {
				t.Fatal("supported-to-unsupported transition compared equal")
			}
		})
	}
}

func TestFingerprintIncompleteMetadata(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"unavailable exports", "unavailable context", "missing AST", "missing declaration", "missing default", "missing parameter name", "missing typed signature", "unsupported semantic subtype", "attached", "panic"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			d := buildFingerprintDriver(t, "public function f(int a = 1) returns int { return a; }")
			ctx, exports, node := d.diagnosticContext(), d.exported, d.pkgNode
			fn := node.Functions[0]
			ref := fn.Symbol()
			switch scenario {
			case "unavailable exports":
				exports = model.ExportedSymbolSpace{}
			case "unavailable context":
				ctx = nil
			case "missing AST":
				node = nil
			case "missing declaration":
				node.Functions = nil
			case "missing default":
				signature, _ := ctx.GetFunctionSignature(ref)
				signature.Default[0] = nil
			case "missing parameter name":
				signature, _ := ctx.GetFunctionSignature(ref)
				signature.ParamNames[0] = ""
			case "missing typed signature":
				ctx.SetFunctionTypedSignature(ref, model.TypedFunctionSignature{})
			case "unsupported semantic subtype":
				signature, _ := ctx.FunctionTypedSignature(ref)
				signature.ParamTypes[0] = semtypes.IntConst(1)
				ctx.SetFunctionTypedSignature(ref, signature)
			case "attached":
				fn.SetAttached()
			case "panic":
				node.Functions = append(node.Functions, nil)
				exports.MainSpaces = append(exports.MainSpaces, nil)
			}
			fp := computeFingerprint(ctx, exports, node)
			if fp.valid || fp.equal(fp) {
				t.Fatalf("incomplete metadata must fail closed: %+v", fp)
			}
		})
	}
}

func TestFingerprintSupportedReuse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, before, after string }{
		{"empty exports", "function f() {}", "function g() {}"},
		{"export order", "public function a() returns int { return 1; } public function b() returns string { return \"b\"; }", "public function b() returns string { return \"b\"; } public function a() returns int { return 1; }"},
		{"default expression", "public function f(int a = 1) returns int { return a; }", "public function f(int a = 2) returns int { return a; }"},
		{"rest body", "public function f(int... a) returns int { _ = a; return 1; }", "public function f(int... a) returns int { _ = a; return 2; }"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a, b := buildFingerprint(t, tc.before), buildFingerprint(t, tc.after)
			if !a.equal(b) {
				t.Fatalf("body/order-only edit must compare equal: %+v, %+v", a, b)
			}
		})
	}
}
