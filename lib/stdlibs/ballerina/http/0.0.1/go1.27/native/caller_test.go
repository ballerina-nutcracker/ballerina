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

// IPv6 loopback is not available on every CI machine, so the address forms are pinned here.
func TestJavaHostAddress(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1":      "127.0.0.1",
		"::1":            "0:0:0:0:0:0:0:1",
		"fe80::1%lo0":    "fe80:0:0:0:0:0:0:1%lo0",
		"2001:db8::ff00": "2001:db8:0:0:0:0:0:ff00",
		"localhost":      "localhost",
	}
	for in, want := range cases {
		if got := javaHostAddress(in); got != want {
			t.Errorf("javaHostAddress(%q) = %q, want %q", in, got, want)
		}
	}
}
