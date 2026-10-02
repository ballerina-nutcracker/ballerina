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


function test((function () returns boolean)[] fs) returns boolean {
    function () returns boolean f = fs[0];
    return f();
}

public function main() {
    int[] xs = [1, 2];
    int[] ys = from int x in xs
        where test([function() returns boolean { string s = 1; return s.length() > x; }]) // @error incompatible type
        select x;
    _ = ys;
}
