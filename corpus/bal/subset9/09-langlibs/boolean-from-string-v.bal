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

// @productions string-literal boolean-literal equality equality-expr local-var-decl-stmt
import ballerina/io;

public function main() {
    io:println(checkpanic 'boolean:fromString("true")); // @output true
    io:println(checkpanic 'boolean:fromString("false")); // @output false
    io:println(checkpanic 'boolean:fromString("TRUE")); // @output true
    io:println(checkpanic 'boolean:fromString("False")); // @output false
    io:println(checkpanic 'boolean:fromString("1")); // @output true
    io:println(checkpanic 'boolean:fromString("0")); // @output false

    boolean|error invalidWord = 'boolean:fromString("yes");
    boolean|error invalidWhitespace = 'boolean:fromString(" true ");
    boolean|error invalidEmpty = 'boolean:fromString("");
    io:println(invalidWord is error); // @output true
    io:println(invalidWhitespace is error); // @output true
    io:println(invalidEmpty is error); // @output true
}
