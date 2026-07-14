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

// @productions error-constructor-expr string-literal exact-equality equality-expr assign-stmt local-var-decl-stmt
import ballerina/io;

public function main() {
    error e1 = error("hi");
    error e2 = error("hi");
    io:println(e1 === e1); // @output true
    io:println(e1 !== e1); // @output false
    io:println(e1 === e2); // @output false
    io:println(e1 !== e2); // @output true
    any|error a1 = e1;
    any|error a2 = e2;
    io:println(a1 === a1); // @output true
    io:println(a1 !== a1); // @output false
    io:println(a1 === a2); // @output false
    io:println(a1 !== a2); // @output true
    error? v1 = e1;
    error? v2 = e2;
    io:println(v1 === v1); // @output true
    io:println(v1 !== v1); // @output false
    io:println(v1 === v2); // @output false
    io:println(v1 !== v2); // @output true
    v2 = ();
    io:println(v1 === v2); // @output false
    io:println(error("hi") === error("hi")); // @output false
}
