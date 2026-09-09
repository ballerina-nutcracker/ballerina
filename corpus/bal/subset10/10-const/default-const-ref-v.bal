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

type Inner record {|
    int x = 1;
    string s = "i";
|};

const Inner I = {x: 2};

type Outer record {|
    Inner inner = I;
    Inner[] list = [I, {}];
|};

const Outer O = {};

public function main() {
    io:println(O); // @output {"inner":{"x":2,"s":"i"},"list":[{"x":2,"s":"i"},{"x":1,"s":"i"}]}
    io:println(O.inner == I); // @output true
    Outer o = {};
    io:println(o == O); // @output true
}
