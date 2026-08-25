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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

function acceptInt(int value) returns string { _ = value; return "int"; }
function acceptString(string value) returns string { _ = value; return "string"; }
function getValue(boolean useInt) returns int|string { return useInt ? 10 : "ten"; }

function classify(int|string value) returns string {
    return value is int ? acceptInt(value) : acceptString(value);
}

function nested(int|string|boolean value) returns string {
    return value is int ? acceptInt(value) : value is string ? acceptString(value) : "boolean";
}

public function main() {
    int|string value = getValue(true);
    int intValue = value is int ? value : 0;
    string stringValue = value is int ? "" : value;
    int negated = !(value is string) ? value : 0;
    io:println(intValue); // @output 10
    io:println(stringValue); // @output
    io:println(negated); // @output 10
    io:println(classify(1)); // @output int
    io:println(classify("one")); // @output string
    io:println(nested(true)); // @output boolean

    boolean flag = true;
    if flag ? (value is int) : (value is int) {
        int narrowedByTernary = value;
        io:println("ternary condition"); // @output ternary condition
        _ = narrowedByTernary;
    }
    if (value is int) ?: false {
        int narrowedByNonNilLhs = value;
        _ = narrowedByNonNilLhs;
    }
    if () ?: value is int {
        int narrowedByRhs = value;
        _ = narrowedByRhs;
    }
    _ = (flag ? (value is int) : ()) ?: value is int;
    if value is int {
        int singleton = true ? value : 0;
        io:println(singleton); // @output 10
    }
}
