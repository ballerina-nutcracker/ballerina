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
int g = 0;

function nonIso() returns int => 1;

public function main() {
    int m = 1;
    isolated function () returns int f1 = () => g; // @error
    isolated function () returns int f2 = () => m; // @error
    isolated function () returns int f3 = () => nonIso(); // @error
    (isolated function () returns int)|int f4 = () => g; // @error
    isolated function () returns (isolated function () returns int) f5 = () => () => g; // @error
    m = 2;
    isolated function () returns int f6 = f5();
    _ = f1() + f2() + f3() + f6();
    _ = f4;
}
