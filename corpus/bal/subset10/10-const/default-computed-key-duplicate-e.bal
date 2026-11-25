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

const K = "a";

// The default is a constant expression, so it is folded and the repeated key
// is reported even though every construction of R supplies m.
type R record {|
    map<int> m = {"a": 1, [K]: 2}; // @error
|};

annotation map<int> Ann on function;

// The value of a non-const annotation is folded too.
@Ann {"a": 1, ["a"]: 2} // @error
function f() {
}

public function main() {
    R r = {m: {}};
    _ = r;
    f();
}
