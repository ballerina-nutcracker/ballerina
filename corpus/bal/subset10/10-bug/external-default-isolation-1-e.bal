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

function next() returns int {
    counter += 1;
    return counter;
}

isolated function dep(int y = next(), typedesc<anydata> td = <>) returns td = external; // @error

isolated function depVar(int y = counter, typedesc<anydata> td = <>) returns td = external; // @error

isolated function ext(int y = next()) returns int = external; // @error

isolated function extVar(int y = counter) returns int = external; // @error

function nonIsolatedDep(int y = next(), typedesc<anydata> td = <>) returns td = external;

function nonIsolatedExt(int y = next()) returns int = external;

class C {
    isolated function dep(int y = next(), typedesc<anydata> td = <>) returns td = external; // @error

    isolated function ext(int y = next()) returns int = external; // @error

    function nonIsolatedExt(int y = next()) returns int = external;
}

isolated function lambdaCapture(int[] xs, isolated function () returns int g = isolated function () returns int {
            return xs.length(); // @error
        }) returns int {
    return g();
}

public function main() {
}
