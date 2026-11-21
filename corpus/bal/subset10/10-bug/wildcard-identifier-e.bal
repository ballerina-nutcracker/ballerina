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

// @labels compound-assignment-stmt local-init-var-decl-stmt function-call-expr
public function main() {
    _ += 1; // @error
    int x = 1;
    x += _; // @error
    int y = _; // @error
    foo(_); // @error
    _ a = 1; // @error
    any b = 1;
    int c = <_>b; // @error
    int|_ d = 1; // @error
    var e = new _(); // @error
    function (int) returns int f = _ => 1; // @error
}

function foo(int a) {
}

function bar() returns _ { // @error
    return 1;
}

type R record {
    *_; // @error
};

enum E {
    _ // @error
}
