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

// @productions type-cast-expr optional-type-descriptor union-type-descriptor nil-type-descriptor local-var-decl-stmt floating-point-literal int-literal boolean-literal function-call-expr
import ballerina/io;

public function main() {
    float f = 2.7;
    int? n = <int?>f;
    io:println(n); // @output 3

    anydata a = 2.7;
    int|string s = <int|string>a;
    io:println(s); // @output 3

    any b = 200.0;
    byte|string bs = <byte|string>b;
    io:println(bs); // @output 200

    decimal d = 3.5d;
    float? fl = <float?>d;
    io:println(fl); // @output 3.5

    any i = 7;
    decimal|boolean db = <decimal|boolean>i;
    io:println(db); // @output 7

    any af = 2.7;
    io:println(<int|float|boolean|()>af); // @output 2.7

    any ai = 5;
    io:println(<int|float|boolean|()>ai); // @output 5

    any ab = true;
    io:println(<int|float|boolean|()>ab); // @output true

    any an = ();
    io:println(<int|float|boolean|()>an is ()); // @output true
}
