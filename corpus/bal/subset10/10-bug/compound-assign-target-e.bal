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

const C = 1;

type T int;

function f() {
}

function byteParamAdd(byte p) {
    p += 1; // @error cannot assign to parameter
}

function nilableParamAdd(int? p) {
    p += 1; // @error cannot assign to parameter
}

public function main() {
    C += 1; // @error cannot assign to constant
    T += 1; // @error cannot assign to type
    f += 1; // @error cannot assign to function
    byteParamAdd(1);
    nilableParamAdd(1);
}
