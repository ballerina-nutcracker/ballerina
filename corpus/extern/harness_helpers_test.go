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
	"flag"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/platform/palnative"
	"ballerina-lang-go/test_util"
	"ballerina-lang-go/test_util/testharness"
)

const (
	testDataDir      = "testdata"
	expectedDir      = "testdata/expected"
	projectExpectDir = "testdata/expected/project"
)

var update = flag.Bool("update", false, "update extern test golden files")

// fileCase builds a TestCase for a single-file test under testdata/.
func fileCase(name string) test_util.TestCase {
	return test_util.TestCase{
		Name:         name,
		InputPath:    filepath.Join(testDataDir, name+".bal"),
		ExpectedPath: filepath.Join(expectedDir, name+".txtar"),
	}
}

// projectCase builds a TestCase for a project test under testdata/.
func projectCase(name string) test_util.TestCase {
	return test_util.TestCase{
		Name:         name,
		InputPath:    filepath.Join(testDataDir, name),
		ExpectedPath: filepath.Join(projectExpectDir, name+".txtar"),
		IsProject:    true,
	}
}

// runExtern wires the harness for one extern test case.
func runExtern(t *testing.T, tc test_util.TestCase, pal testharness.TestPal, externs []testharness.ExternRegistration) {
	t.Helper()
	testharness.Run(t, tc, pal, externs)
	if *update {
		testharness.Update(t, tc, pal)
		return
	}
	testharness.Validate(t, tc, pal)
}

// httpPal wraps a default in-memory TestPal but overrides Platform() to inject
// a caller-supplied HTTP client factory. Stdout/stderr/diagnostics still flow
// through the embedded TestPal. When realFS is true, FS.ReadFile delegates
// to os.ReadFile (used by tests that need to load cert files from disk).
type httpPal struct {
	testharness.TestPal
	newClient func(cfg pal.ClientConfig) pal.HTTPClient
	realFS    bool
}

// newHTTPPal returns a TestPal whose Platform()'s HTTP.NewClient is overridden.
func newHTTPPal(newClient func(cfg pal.ClientConfig) pal.HTTPClient) *httpPal {
	return &httpPal{TestPal: testharness.NewTestPal(), newClient: newClient}
}

// withRealFS returns a copy of p whose FS.ReadFile delegates to os.ReadFile.
func (p *httpPal) withRealFS() *httpPal {
	cp := *p
	cp.realFS = true
	return &cp
}

func (p *httpPal) Platform() pal.Platform {
	base := p.TestPal.Platform()
	base.HTTP = pal.HTTP{NewClient: p.newClient, Listen: palnative.Listen}
	if p.realFS {
		base.FS = pal.FS{
			ReadFile: os.ReadFile,
			OpenReadable: func(path string) (io.ReadCloser, error) {
				return os.Open(path)
			},
			OpenWritable: func(path string, appendMode bool) (io.WriteCloser, error) {
				openFlag := os.O_CREATE | os.O_WRONLY
				if appendMode {
					openFlag |= os.O_APPEND
				} else {
					openFlag |= os.O_TRUNC
				}
				return os.OpenFile(path, openFlag, 0o644)
			},
		}
	}
	return base
}

// skipIfNoNetwork skips the test when EXTERN_SKIP_NETWORK is set or when
// running under WASM (js/wasm), which has no outbound TCP access.
func skipIfNoNetwork(t *testing.T) {
	t.Helper()
	if os.Getenv("EXTERN_SKIP_NETWORK") != "" || goruntime.GOOS == "js" {
		t.Skip("skipping network-dependent test")
	}
}
