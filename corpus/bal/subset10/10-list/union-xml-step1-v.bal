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

import ballerina/io;

function show(any v) {
    io:println(v is [xml, xml, xml, int]); // @output true
    io:println(v is [xml, xml, xml, string]); // @output false
}

public function main() {
    xml x = xml `<root><a>1</a><a>2</a></root>`;
    [xml, xml, xml, int]|[xml, xml, xml, string] l = [x/<a>, x/<a>[1], x/<a>.get(0), 1];
    show(l);
    io:println(l); // @output [<a>1</a><a>2</a>,<a>2</a>,<a>1</a>,1]
}
