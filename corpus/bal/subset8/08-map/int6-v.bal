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

// @productions is-expr map-type-descriptor mapping-constructor-expr string string-literal if-else-stmt any local-var-decl-stmt
import ballerina/io;

public function main() {
    map<int> im = {};
    if im is map<any> {
        io:println("map<any>"); // @output map<any>
    }
    if im is map<string> {
        io:println("map<string>");
    }
    if im is map<int?> {
        io:println("map<int?>"); // @output map<int?>
    }
    if im is map<int|string> {
        io:println("map<int|string>"); // @output map<int|string>
    }
}
