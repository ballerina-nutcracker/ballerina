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

class ExpFn {
    function init(function (int) returns int callback, int marker) {
        var _ = callback;
        var _ = marker;
    }
}

class ExpStr {
    function init(function (string) returns int callback, int marker) {
        var _ = callback;
        var _ = marker;
    }
}

function apply(function () returns int f) returns int {
    return f();
}

public function main() {
    // Known limitation, see "New expression" in doc/lang/subset10.md: the explicitly typed anonymous function is
    // checked from its signature only while selecting a candidate, so the string candidate is not ruled out.
    ExpFn|ExpStr m = new (v => apply(function () returns int { return v + 1; }), 1); // @error
    var _ = m;
}
