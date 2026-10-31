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

import ballerina/data.jsondata;
import ballerina/io;

// jBallerina's compiler plugin rejects both targets below at compile time: a record whose `@jsondata:Name`
// overrides clash (JSON_ERROR_202) and an xml target for a parse function (JSON_ERROR_203). Without
// compiler-plugin support the same checks run when the type is used as a parse target.
type Clash record {|
    @jsondata:Name {value: "b"}
    string a;
    string b;
|};

type Markup xml;

function show(anydata|error v) {
    if v is error {
        io:println("error: ", v.message());
    } else {
        io:println(v);
    }
}

public function main() {
    Clash|error c1 = jsondata:parseString(string `{"b":"x"}`, {}, Clash);
    show(c1);
    // @output error: duplicate field 'b'
    Clash|error c2 = jsondata:parseAsType({b: "x"}, {}, Clash);
    show(c2);
    // @output error: duplicate field 'b'
    Markup|error m1 = jsondata:parseString("1", {}, Markup);
    show(m1);
    // @output error: unsupported type 'xml'
    Markup|error m2 = jsondata:parseAsType(1, {}, Markup);
    show(m2);
    // @output error: invalid type 'xml' expected 'anydata'
}
