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

type R record {|
    U? a;
|};

type U R|map<U>|U[];

function rep(string x, int n) returns string {
    if n == 0 {
        return "";
    }
    string h = rep(x, n / 2);
    return n % 2 == 0 ? h + h : h + h + x;
}

public function main() {
    // R names the field it converts and map<U> does not, so each node is reached with every name count.
    string invalid = rep("{\"a\":", 2000) + "{\"z\":1}" + rep("}", 2000);
    U|error u1 = jsondata:parseString(invalid);
    io:println(u1 is jsondata:Error);
    // @output true
    json|error invalidJson = jsondata:parseString(invalid);
    if invalidJson is json {
        U|error u2 = jsondata:parseAsType(invalidJson);
        io:println(u2 is jsondata:Error);
    }
    // @output true
}
