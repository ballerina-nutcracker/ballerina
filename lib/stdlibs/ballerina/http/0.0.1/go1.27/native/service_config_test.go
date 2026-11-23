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

package native

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// http-svc-service-config-v covers host-based dispatch; the module's client always sends a port and IPv6
// loopback is not available on every CI machine, so these two Host forms are pinned here.
func TestHostServices(t *testing.T) {
	t.Parallel()
	entry := func(host string) *serviceEntry { return &serviceEntry{config: serviceConfig{host: host}} }
	defaultEntry, byName, ipv6 := entry(""), entry("localhost"), entry("::1")
	services := []*serviceEntry{defaultEntry, byName, ipv6}
	for _, c := range []struct {
		requestHost string
		want        *serviceEntry
	}{
		{"localhost", byName},
		{"[::1]:9090", ipv6},
	} {
		if got := hostServices(services, c.requestHost); len(got) != 1 || got[0] != c.want {
			t.Errorf("hostServices(%q) = %v, want the %q service", c.requestHost, got, c.want.config.host)
		}
	}
}

// http-svc-service-config-v covers the prefix rules; the module's client does not see a 101 as sent, and the
// status count is not visible to a client, so the writer is checked directly for both.
func TestSubtypePrefixWriter(t *testing.T) {
	cases := []struct {
		contentType string
		status      int
		want        string
	}{
		{"application/json", http.StatusOK, "application/vnd.x+json"},
		{"text/plain", http.StatusSwitchingProtocols, "text/vnd.x+plain"},
	}
	for _, c := range cases {
		recorder := &countingRecorder{ResponseRecorder: httptest.NewRecorder()}
		w := &subtypePrefixWriter{ResponseWriter: recorder, prefix: "vnd.x"}
		w.Header().Set("Content-Type", c.contentType)
		w.WriteHeader(c.status)
		// A 101 has no body; the write must not send a second status either.
		if _, err := w.Write([]byte("x")); err != nil && c.status != http.StatusSwitchingProtocols {
			t.Fatal(err)
		}
		if got := recorder.Header().Get("Content-Type"); got != c.want {
			t.Errorf("Content-Type for %q = %q, want %q", c.contentType, got, c.want)
		}
		if recorder.Code != c.status || recorder.writeHeaderCalls != 1 {
			t.Errorf("status for %q = %d after %d WriteHeader calls, want %d once", c.contentType, recorder.Code,
				recorder.writeHeaderCalls, c.status)
		}
	}
}

// countingRecorder counts WriteHeader calls, which httptest.ResponseRecorder drops after the first.
type countingRecorder struct {
	*httptest.ResponseRecorder
	writeHeaderCalls int
}

func (r *countingRecorder) WriteHeader(statusCode int) {
	r.writeHeaderCalls++
	r.ResponseRecorder.WriteHeader(statusCode)
}

// A ResponseWriter sends an implicit 200 on the first Write, so the prefix must be added then too. Every
// module path sets the status before the body, so a corpus test cannot reach this.
func TestSubtypePrefixWriterImplicitHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	w := &subtypePrefixWriter{ResponseWriter: recorder, prefix: "vnd.x"}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte("{}")); err != nil {
		t.Fatal(err)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/vnd.x+json" {
		t.Errorf("Content-Type = %q", got)
	}
}
