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

isolated function next() returns int {
    return 42;
}

type Runtime record {|
    int x = next();
    int y = 1;
|};

type Inner record {|
    int x = next();
|};

type Outer record {|
    Inner inner = {};
    int y = 2;
|};

const Runtime A = {x: 7};
const Outer B = {inner: {x: 1}};

public function main() {
    io:println(A); // @output {"x":7,"y":1}
    io:println(B); // @output {"inner":{"x":1},"y":2}
    Runtime r = {};
    io:println(r); // @output {"x":42,"y":1}
    Outer o = {};
    io:println(o); // @output {"inner":{"x":42},"y":2}
}
