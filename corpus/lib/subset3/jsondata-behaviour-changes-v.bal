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

// Inputs on which jBallerina panics (see "Notable Behavioural Changes" in the data.jsondata README); this
// port returns a jsondata:Error for each of them instead.
type Pair record {|
    int a;
    int b;
|};

function show(anydata|error v) {
    if v is error {
        io:println(v is jsondata:Error, " ", v.message());
    } else {
        io:println(v);
    }
}

public function main() {
    stream<byte[], error?> s1 = ["{\"a\":\"x\"}".toBytes()].toStream();
    Pair|jsondata:Error r1 = jsondata:parseStream(s1);
    show(r1);
    // @output true incompatible value 'x' for type 'int' in field 'a'
    json|jsondata:Error r2 = jsondata:parseString("[1,,2]");
    show(r2);
    // @output true incompatible expected type 'json' for value ''
    decimal|jsondata:Error r3 = jsondata:parseString("1e6145");
    show(r3);
    // @output true incompatible expected type 'decimal' for value '1e6145'
    json|jsondata:Error r4 = jsondata:parseString("{\"a\":");
    show(r4);
    // @output true invalid JSON document
    Pair|jsondata:Error r5 = jsondata:parseString("{}");
    show(r5);
    // @output true required field 'a' not present in JSON
    string|jsondata:Error r6 = jsondata:parseString("\"\\ud83d|\\ud83d\\ude00\"");
    if r6 is string {
        io:println(r6.toBytes().toBase16()); // @output efbfbd7cf09f9880
    }
}
