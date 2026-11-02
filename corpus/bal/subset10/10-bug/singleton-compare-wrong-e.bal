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
import ballerina/io;

const int MIN = 1;
const false C = MIN < 2; // @error C is true

public function main() {
    false a = 1 < 2; // @error result is true
    true b = "b" <= "a"; // @error result is false
    true c = 1.0d != 1.00d; // @error result is false
    true d = float:NaN < float:NaN; // @error result is false
    false e = float:NaN == float:NaN; // @error result is true
    io:println(a, b, c, d, e, C);
}
