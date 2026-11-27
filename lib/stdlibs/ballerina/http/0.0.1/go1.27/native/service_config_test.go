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

import "testing"

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
