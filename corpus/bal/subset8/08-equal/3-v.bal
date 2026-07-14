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

// @productions mapping-constructor-expr list-constructor-expr exact-equality equality-expr any function-call-expr local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    any x = {v: 1};
    any y = x;
    // @output true
    // @output false
    printEq(x, y);
    // @output false
    // @output true
    printEq({v: 1}, 1);
    // @output false
    // @output true
    printEq({v: 1}, [1]);
    // @output false
    // @output true
    printEq({v: 1}, {v: 1});
    // @output false
    // @output true
    printEq({}, {});
    // @output false
    // @output true
    printEq({v: 1}, {v: 2});
}

function printEq(any x, any y) {
    io:println(x === y);
    io:println(x !== y);
}
