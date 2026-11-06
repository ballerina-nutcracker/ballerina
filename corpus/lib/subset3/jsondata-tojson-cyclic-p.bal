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

public function main() {
    map<anydata> m = {};
    m["self"] = m;
    any|error trapped = trap jsondata:toJson(m);
    io:println(trapped is jsondata:Error); // @output true

    anydata[] list = [1];
    list.push(list);
    json result = jsondata:toJson(list); // @panic the value has a cyclic reference
    io:println(result);
}
