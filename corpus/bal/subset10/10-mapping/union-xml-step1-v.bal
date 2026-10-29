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

type A record {|
    xml v;
    xml first;
    xml got;
    int count;
    int m;
|};

type B record {|
    xml v;
    xml first;
    xml got;
    int count;
    string m;
|};

function show(any v) {
    io:println(v is A); // @output true
    io:println(v is B); // @output false
}

public function main() {
    xml x = xml `<root><a>1</a><a>2</a></root>`;
    A|B m = {v: x/<a>, first: x/<a>[0], got: x/<a>.get(1), count: (x/<a>).length(), m: 1};
    show(m);
    io:println(m); // @output {"v":<a>1</a><a>2</a>,"first":<a>1</a>,"got":<a>2</a>,"count":2,"m":1}
}
