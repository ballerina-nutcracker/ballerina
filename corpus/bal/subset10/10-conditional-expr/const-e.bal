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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

boolean runtimeCondition = true;
int runtimeValue = 10;
const A = runtimeCondition ? 1 : 2; // @error
const B = true ? runtimeValue : 2; // @error
const C = true ? 1 : runtimeValue; // @error
const D = runtimeValue ?: 1; // @error
const E = () ?: runtimeValue; // @error
const F = 1 ?: runtimeValue; // @error

public function main() {
    _ = A;
    _ = B;
    _ = C;
    _ = D;
    _ = E;
    _ = F;
}
