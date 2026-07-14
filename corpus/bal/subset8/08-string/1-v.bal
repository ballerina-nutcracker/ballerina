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

// @productions string string-literal equality boolean if-else-stmt equality-expr boolean-literal return-stmt function-call-expr
import ballerina/io;

public function main() {
    io:println(isKeyword("string")); // @output true
    io:println(isKeyword("hello")); // @output false
    io:println(isKeyword("if")); // @output true
    io:println(isKeyword("else")); // @output true
    io:println(isKeyword("false")); // @output true
    io:println(isKeyword("return")); // @output true
}

function isKeyword(string s) returns boolean {
    if s == "return" {
        return true;
    }
    if s == "boolean" {
        return true;
    }
    if s == "int" {
        return true;
    }
    if s == "string" {
        return true;
    }
    if s == "while" {
        return true;
    }
    if s == "foreach" {
        return true;
    }
    if s == "if" {
        return true;
    }
    if s == "else" {
        return true;
    }
    if s == "map" {
        return true;
    }
    if s == "true" {
        return true;
    }
    if s == "false" {
        return true;
    }
    return false;
}
