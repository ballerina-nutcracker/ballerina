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
import ballerina/file;
import ballerina/io;

type Row record {|int a; string b;|};

function printRows(string log) returns error? {
    foreach string line in check io:fileReadLines(log) {
        int? i = line.indexOf("\"location\"");
        io:println(i is int ? line.substring(i) : line);
    }
}

public function main() returns error? {
    string dir = check file:createTempDir();

    // Every skipped line still counts towards the row of the later errors.
    string log1 = check file:joinPath(dir, "rows1.log");
    csv:FailSafeOptions fs1 = {enableConsoleLogs: false, fileOutputMode: {filePath: log1, contentType: csv:METADATA}};
    Row[]|csv:Error rows = csv:parseString("a,b\nx,1\ny,2\nz,3", {failSafe: fs1});
    io:println(rows);
    check printRows(log1);

    string log2 = check file:joinPath(dir, "rows2.log");
    csv:FailSafeOptions fs2 = {enableConsoleLogs: false, fileOutputMode: {filePath: log2, contentType: csv:METADATA}};
    rows = csv:parseString("a,b\n1,x\nx,1,2\n2,y\ny,\\q\n3,z", {failSafe: fs2});
    io:println(rows);
    check printRows(log2);

    check file:remove(dir, file:RECURSIVE);
}

// @output []
// @output "location":{"row":2,"column":1},"message":"value 'x' cannot be cast into 'int'"}
// @output "location":{"row":3,"column":1},"message":"value 'y' cannot be cast into 'int'"}
// @output "location":{"row":4,"column":1},"message":"value 'z' cannot be cast into 'int'"}
// @output [{"a":1,"b":"x"},{"a":2,"b":"y"},{"a":3,"b":"z"}]
// @output "location":{"row":3,"column":1},"message":"value 'x' cannot be cast into 'int'"}
// @output "location":{"row":5,"column":1},"message":"value 'y' cannot be cast into 'int'"}
