// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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
import ballerina/protobuf.types.wrappers;

public function main() {
    wrappers:ContextBoolean ctxBool = {content: true, headers: {}};
    io:println(ctxBool.content); // @output true

    wrappers:ContextBytes ctxBytes = {content: [1, 2, 3], headers: {}};
    io:println(ctxBytes.content); // @output [1,2,3]

    wrappers:ContextFloat ctxFloat = {content: 1.5, headers: {}};
    io:println(ctxFloat.content); // @output 1.5

    wrappers:ContextInt ctxInt = {content: 42, headers: {}};
    io:println(ctxInt.content); // @output 42

    wrappers:ContextString ctxStr = {content: "hello", headers: {}};
    io:println(ctxStr.content); // @output hello
}
