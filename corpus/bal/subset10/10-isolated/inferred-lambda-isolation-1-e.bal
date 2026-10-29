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

int counter = 0;

type IsolatedFn isolated function (int) returns int;

function nonIsolated(int x) returns int {
    counter += x;
    return x;
}

function apply(isolated function (int) returns int f) returns int {
    return f(1);
}

// A lambda with inferred parameters is isolated when its expected function
// type is isolated, so its body is checked like any isolated function's.
function assignedToIsolatedType() {
    IsolatedFn g = v => nonIsolated(v); // @error calls a non-isolated function
    _ = g(1);
}

function passedToIsolatedParam() {
    _ = apply(x => x + counter + 1); // @error reads mutable module state
}

final IsolatedFn moduleLevel = v => v + counter; // @error reads mutable module state

function defaultParam(IsolatedFn f = x => x + counter) returns int { // @error reads mutable module state
    return f(0);
}

function capturesMutableLocal() {
    int[] arr = [1];
    IsolatedFn g = v => v + arr[0]; // @error captures a mutable local
    _ = g(1);
}

public function main() {
    assignedToIsolatedType();
    passedToIsolatedParam();
    _ = moduleLevel(1);
    _ = defaultParam();
    capturesMutableLocal();
}
