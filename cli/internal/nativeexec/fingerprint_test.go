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

package nativeexec

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestFingerprintPayloads_Deterministic(t *testing.T) {
	t.Parallel()
	p := &GoSourcePayload{
		GoFiles: fstest.MapFS{"a.go": {Data: []byte("package a")}},
		Module:  "example.com/a",
	}
	fp1, err := FingerprintPayloads([]NativePayload{p})
	if err != nil {
		t.Fatal(err)
	}
	fp2, err := FingerprintPayloads([]NativePayload{p})
	if err != nil {
		t.Fatal(err)
	}
	if fp1 != fp2 {
		t.Errorf("FingerprintPayloads not deterministic: %q vs %q", fp1, fp2)
	}
}

func TestFingerprintPayloads_OrderIndependent(t *testing.T) {
	t.Parallel()
	pa := &GoSourcePayload{
		GoFiles: fstest.MapFS{"a.go": {Data: []byte("package a")}},
		Module:  "example.com/a",
	}
	pb := &GoSourcePayload{
		GoFiles: fstest.MapFS{"b.go": {Data: []byte("package b")}},
		Module:  "example.com/b",
	}
	fp1, err := FingerprintPayloads([]NativePayload{pa, pb})
	if err != nil {
		t.Fatal(err)
	}
	fp2, err := FingerprintPayloads([]NativePayload{pb, pa})
	if err != nil {
		t.Fatal(err)
	}
	if fp1 != fp2 {
		t.Errorf("FingerprintPayloads is order-dependent: %q vs %q", fp1, fp2)
	}
}

func TestFingerprintPayloads_SeedInfluence(t *testing.T) {
	t.Parallel()
	p := &GoSourcePayload{
		GoFiles: fstest.MapFS{"a.go": {Data: []byte("package a")}},
		Module:  "example.com/a",
	}
	fpNoSeed, err := FingerprintPayloads([]NativePayload{p})
	if err != nil {
		t.Fatal(err)
	}
	fpWithSeed, err := FingerprintPayloads([]NativePayload{p}, []byte("go1.26.0 linux/amd64"))
	if err != nil {
		t.Fatal(err)
	}
	if fpNoSeed == fpWithSeed {
		t.Error("seed must change the fingerprint")
	}
}

func TestFingerprintPayloads_ContentChange(t *testing.T) {
	t.Parallel()
	p1 := &GoSourcePayload{
		GoFiles: fstest.MapFS{"a.go": {Data: []byte("package a")}},
		Module:  "example.com/a",
	}
	p2 := &GoSourcePayload{
		GoFiles: fstest.MapFS{"a.go": {Data: []byte("package a // changed")}},
		Module:  "example.com/a",
	}
	fp1, err := FingerprintPayloads([]NativePayload{p1})
	if err != nil {
		t.Fatal(err)
	}
	fp2, err := FingerprintPayloads([]NativePayload{p2})
	if err != nil {
		t.Fatal(err)
	}
	if fp1 == fp2 {
		t.Error("file content change must change the fingerprint")
	}
}

func TestFingerprintPayloads_NonGoFilesIgnored(t *testing.T) {
	t.Parallel()
	p1 := &GoSourcePayload{
		GoFiles: fstest.MapFS{"a.go": {Data: []byte("package a")}},
		Module:  "example.com/a",
	}
	p2 := &GoSourcePayload{
		GoFiles: fstest.MapFS{
			"a.go":      {Data: []byte("package a")},
			"README.md": {Data: []byte("changed readme content")},
		},
		Module: "example.com/a",
	}
	fp1, err := FingerprintPayloads([]NativePayload{p1})
	if err != nil {
		t.Fatal(err)
	}
	fp2, err := FingerprintPayloads([]NativePayload{p2})
	if err != nil {
		t.Fatal(err)
	}
	if fp1 != fp2 {
		t.Error("non-.go file changes must not affect fingerprint")
	}
}

func TestFingerprintPayloads_EmptyPayloads(t *testing.T) {
	t.Parallel()
	fp, err := FingerprintPayloads(nil)
	if err != nil {
		t.Fatal(err)
	}
	if fp == "" {
		t.Error("fingerprint must not be empty even for empty payloads")
	}
}

func TestFingerprintPath(t *testing.T) {
	t.Parallel()
	got := FingerprintPath("/path/to/bin/bal")
	want := "/path/to/bin/bal.fingerprint"
	if got != want {
		t.Errorf("FingerprintPath = %q, want %q", got, want)
	}
}

func TestWriteFingerprint_CreatesFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "bal")

	if err := WriteFingerprint(binPath, "abc123"); err != nil {
		t.Fatalf("WriteFingerprint: %v", err)
	}
	data, err := os.ReadFile(FingerprintPath(binPath))
	if err != nil {
		t.Fatalf("reading fingerprint file: %v", err)
	}
	if string(data) != "abc123" {
		t.Errorf("fingerprint content = %q, want %q", string(data), "abc123")
	}
}

func TestWriteFingerprint_Overwrites(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "bal")

	if err := WriteFingerprint(binPath, "first"); err != nil {
		t.Fatal(err)
	}
	if err := WriteFingerprint(binPath, "second"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(FingerprintPath(binPath))
	if string(data) != "second" {
		t.Errorf("second write must overwrite: got %q", string(data))
	}
}

func TestWriteFingerprint_MissingDir(t *testing.T) {
	t.Parallel()
	err := WriteFingerprint("/nonexistent/dir/bal", "fp")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}
