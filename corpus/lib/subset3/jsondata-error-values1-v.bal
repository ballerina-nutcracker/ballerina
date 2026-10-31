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

function show(anydata|error v) {
    if v is error {
        io:println("error: ", v.message());
    } else {
        io:println(v);
    }
}

public function main() {
    boolean|string|error a = jsondata:parseAsType(1234567);
    show(a);
    // @output error: incompatible expected type '(boolean|string)' for value '1,234,567'
    boolean|string|error b = jsondata:parseAsType(-12345.67891);
    show(b);
    // @output error: incompatible expected type '(boolean|string)' for value '-12,345.679'
    boolean|string|error c = jsondata:parseAsType(0.0005);
    show(c);
    // @output error: incompatible expected type '(boolean|string)' for value '0.001'
    boolean|string|error d = jsondata:parseAsType(2.0015);
    show(d);
    // @output error: incompatible expected type '(boolean|string)' for value '2.002'
    boolean|string|error e = jsondata:parseAsType(999.9996);
    show(e);
    // @output error: incompatible expected type '(boolean|string)' for value '1,000'
    boolean|string|error f = jsondata:parseAsType(-9223372036854775807 - 1);
    show(f);
    // @output error: incompatible expected type '(boolean|string)' for value '-9,223,372,036,854,775,808'
    boolean|string|error g = jsondata:parseAsType(1.0e20);
    show(g);
    // @output error: incompatible expected type '(boolean|string)' for value '100,000,000,000,000,000,000'
    boolean|string|error h = jsondata:parseAsType(12);
    show(h);
    // @output error: incompatible expected type '(boolean|string)' for value '12'
}
