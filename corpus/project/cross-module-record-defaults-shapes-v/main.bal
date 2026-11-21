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

import crossmodulerecorddefaultsshapes.base;
import ballerina/io;

const base:Shapes A = {};
const base:Shapes B = {flag: true};

public function main() {
    io:println(A); // @output {"f":2.0,"d":1.5,"b":255,"flag":false,"mode":"off","hidden":7,"extra":{"h":3}}
    io:println(A.f is float); // @output true
    io:println(A.d is decimal); // @output true
    io:println(A.b is byte); // @output true
    io:println(A.extra is readonly); // @output true
    io:println(B); // @output {"flag":true,"f":2.0,"d":1.5,"b":255,"mode":"off","hidden":7,"extra":{"h":3}}
    base:Shapes r = {};
    io:println(r == A); // @output true
}
