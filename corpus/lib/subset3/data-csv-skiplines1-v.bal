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

import ballerina/data.csv;
import ballerina/io;

type Row record {|int a; string b;|};

type Rows Row[];

public function main() {
    io:println(csv:parseString("a,b\n1,x\n2,y\n3,z\n4,w", {skipLines: [1, 3]}, Rows));
    io:println(csv:parseString("a,b\n1,x\n2,y\n3,z\n4,w", {skipLines: "2-3"}, Rows));
    io:println(csv:parseString("a,b\n1,x\n2,y", {skipLines: [0, -1, 9]}, Rows));
    io:println(csv:parseString("a,b\n1,x\n2,y", {skipLines: "+2-2"}, Rows));
    // Blank and comment lines are not data rows.
    io:println(csv:parseString("a,b\n1,x\n\n# c\n2,y\n3,z", {skipLines: [2]}, Rows));
    // Invalid ranges.
    io:println(csv:parseString("a,b\n1,x", {skipLines: "3-1"}, Rows));
    io:println(csv:parseString("a,b\n1,x", {skipLines: "a-1"}, Rows));
    io:println(csv:parseString("a,b\n1,x", {skipLines: "5"}, Rows));
    io:println(csv:parseString("a,b\n1,x", {skipLines: "1-"}, Rows));
    io:println(csv:parseString("a,b\n1,x", {skipLines: "-5"}, Rows));
    // An invalid range is only reported once a data row is read.
    io:println(csv:parseString("a,b", {skipLines: "3-1"}, Rows));
    io:println(csv:transform([{"a": 1, "b": "x"}, {"a": 2, "b": "y"}], {skipLines: [1]}, Rows));
    io:println(csv:transform([{"a": 1, "b": "x"}], {skipLines: "x"}, Rows));
}

// @output [{"a":2,"b":"y"},{"a":4,"b":"w"}]
// @output [{"a":1,"b":"x"},{"a":4,"b":"w"}]
// @output [{"a":1,"b":"x"},{"a":2,"b":"y"}]
// @output [{"a":1,"b":"x"}]
// @output [{"a":1,"b":"x"},{"a":2,"b":"y"},{"a":3,"b":"z"}]
// @output error Error ("invalid range for the skipLines field. Start value must be less than or equal to end value.")
// @output error Error ("invalid input for the skipLines field. Both start and end values must be integers.")
// @output error Error ("invalid range for the skipLines field. Start value must be less than or equal to end value.")
// @output error Error ("invalid range for the skipLines field. Start value must be less than or equal to end value.")
// @output error Error ("invalid input for the skipLines field. Both start and end values must be integers.")
// @output []
// @output [{"a":2,"b":"y"}]
// @output error Error ("invalid range for the skipLines field. Start value must be less than or equal to end value.")
