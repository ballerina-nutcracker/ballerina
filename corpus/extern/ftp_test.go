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
	"testing"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// ftpPal wraps the in-memory TestPal with real TCP dialing so the ftp fixtures can reach the
// in-process servers started per test.
type ftpPal struct {
	testharness.TestPal
}

func (p *ftpPal) Platform() pal.Platform {
	base := p.TestPal.Platform()
	base.Net = pal.Net{Dial: palnative.Dial, ClientTLSConfig: palnative.ClientTLSConfig}
	return base
}

// runFtp runs testdata/ftp/<name>.bal against fresh servers; the fixture reaches them through
// external functions returning their ports and the generated key and certificate paths.
func runFtp(t *testing.T, name string) {
	t.Helper()
	t.Parallel()
	skipIfNoNetwork(t)
	f := startFtpFixture(t)
	value := func(v values.BalValue) extern.NativeFunc {
		return func(*extern.Context, []values.BalValue) (values.BalValue, error) { return v, nil }
	}
	fixtures := map[string]values.BalValue{
		"ftpPort":              f.ftpPort,
		"ftpsPort":             f.ftpsPort,
		"ftpsImplicitPort":     f.ftpsImplicitPort,
		"sftpPort":             f.sftpPort,
		"certPath":             f.certPath,
		"keyPath":              f.keyPath,
		"encryptedKeyPath":     f.encryptedKeyPath,
		"knownHostsPath":       f.knownHostsPath,
		"badKnownHostsPath":    f.badKnownHosts,
		"hashedKnownHostsPath": f.hashedKnownHosts,
		"missingKeyPath":       f.missingKeyPath,
	}
	var externs []testharness.ExternRegistration
	for fn, v := range fixtures {
		externs = append(externs, testharness.ExternRegistration{Org: "$anon", Module: name, FuncName: fn, Impl: value(v)})
	}
	runExtern(t, fileCase("ftp/"+name), &ftpPal{TestPal: testharness.NewTestPal()}, externs)
}

func TestFtpShowcase(t *testing.T) {
	runFtp(t, "ftp-showcase-v")
}

func TestFtpClientOps(t *testing.T) {
	runFtp(t, "ftp-client-ops-v")
}

func TestFtpSecure(t *testing.T) {
	runFtp(t, "ftp-secure-v")
}

func TestFtpStreams(t *testing.T) {
	runFtp(t, "ftp-streams-v")
}

func TestFtpResilience(t *testing.T) {
	runFtp(t, "ftp-resilience-v")
}

func TestFtpListenerContent(t *testing.T) {
	runFtp(t, "ftp-listener-content-v")
}

func TestFtpListenerLegacy(t *testing.T) {
	runFtp(t, "ftp-listener-legacy-v")
}

func TestFtpMisc(t *testing.T) {
	runFtp(t, "ftp-misc-v")
}
