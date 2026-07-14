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

// @productions string-literal exact-equality equality if-else-stmt equality-expr relational-expr int-literal
import ballerina/io;

public function main() {
    // Type of this is not singleton, so no error
    if (0 > 1) == (1 === 1) {
        io:println("true");
    }
    else {
        io:println("false"); // @output false
    }
}
