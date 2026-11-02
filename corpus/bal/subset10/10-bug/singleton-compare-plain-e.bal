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

public function main() {
    true exact = MIN === 1; // @error === is not singleton typed
    1|2 x = 1;
    true union = x < 3; // @error union of singletons is not a singleton
    int y = 1;
    true plain = y < 2; // @error int operand
    int? z = 1;
    true optional = z == 1; // @error optional operand
    io:println(exact, union, plain, optional);
}
