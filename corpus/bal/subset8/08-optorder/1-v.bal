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

// @productions relational-expr local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    int? n1 = 1;
    int? n2 = ();
    io:println(n1 < n2); // @output false
    io:println(n1 <= n2); // @output false
    io:println(n1 > n2); // @output false
    io:println(n1 >= n2); // @output false

    io:println(n2 < n1); // @output false
    io:println(n2 <= n1); // @output false
    io:println(n2 > n1); // @output false
    io:println(n2 >= n1); // @output false

    int? n3 = 5;
    io:println(n1 <= n3); // @output true
    io:println(n1 < n3); // @output true
    io:println(n1 >= n3); // @output false
    io:println(n1 > n3); // @output false

    int n4 = 5;
    io:println(n1 <= n4); // @output true
    io:println(n1 < n4); // @output true
    io:println(n1 >= n4); // @output false
    io:println(n1 > n4); // @output false

    io:println(n4 <= n1); // @output false
    io:println(n4 < n1); // @output false
    io:println(n4 >= n1); // @output true
    io:println(n4 > n1); // @output true

    int? n5 = ();
    io:println(n5 < n2); // @output false
    io:println(n5 <= n2); // @output true
    io:println(n5 > n2); // @output false
    io:println(n5 >= n2); // @output true
}
