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

// @productions equality if-else-stmt equality-expr return-stmt function-call-expr local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    if makeNil() == null {
        io:println(1); // @output 1
    }
    if null == makeNil() {
        io:println(2); // @output 2
    }
    if null == null {
        io:println(3); // @output 3
    }
    if makeNil() != null {
    }
    else {
        io:println(4); // @output 4
    }
    if null != makeNil() {
    }
    else {
        io:println(5); // @output 5
    }
    boolean b = null != null;
    if b {
        io:println(6);
    }
}

function makeNil() {
    return;
}
