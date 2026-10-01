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


function call(function () returns int f) returns int => f();

function ret() returns int {
    return 1 + call(function() returns int { string s = 1; return s.length(); }); // @error incompatible type
}

public function main() {
    int x = 0;
    x = 1 + call(function() returns int { string s = 2; return s.length(); }); // @error incompatible type
    x += 1 + call(function() returns int { string s = 3; return s.length(); }); // @error incompatible type
    int[] xs = [1];
    xs[call(function() returns int { string s = 4; return s.length(); })] = 1; // @error incompatible type
    _ = ret() + x;
}
