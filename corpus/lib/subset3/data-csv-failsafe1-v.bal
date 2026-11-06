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

type Rows Row[];

public function main() returns error? {
    // Tolerated errors are logged to the console and the offending row is skipped.
    io:println(csv:parseString("a,b\n1,x\nz,y\n2,w", {failSafe: {}}, Rows));
    io:println(csv:parseString("a,b\n1,x\nz,y\n2,w", {failSafe: {includeSourceDataInConsole: true}}, Rows));
    io:println(csv:parseString("a,b\n1,x\n2,C:\\x\n3,w", {failSafe: {}}, Rows));
    Rows|csv:Error reopened = csv:parseString("a,b\n1,x\n\"2\",\"y\" ,\"q\nz,1\n3,w", {failSafe: {}}, Rows);
    io:println(reopened is Rows && reopened.length() == 2 && reopened[1].b == "y\" ,\"q\nz,1\n3,w");
    io:println(csv:parseString("a,b\n1,x\nz,y", {failSafe: {}}, Rows));
    io:println(csv:parseString("a,b\n1,x\n2,\\x", {failSafe: {}}, Rows));
    // A row missing a required field is logged and skipped.
    io:println(csv:parseString("a,b\n1,x\n2\n3,w", {failSafe: {}}, Rows));
    // Some errors are never tolerated.
    io:println(csv:parseString("", {failSafe: {}}, Rows));
    // transform and parseList never log to the console.
    io:println(csv:transform([{"a": "z", "b": "y"}, {"a": 1, "b": "q"}], {failSafe: {}}, Rows));
    io:println(csv:parseList([["z", "y"], ["1", "q"]], {customHeaders: ["a", "b"], failSafe: {}}, Rows));

    // Errors can also be written to a file.
    string dir = check file:createTempDir();
    string rawLog = check file:joinPath(dir, "logs", "raw.log");
    csv:FailSafeOptions raw = {enableConsoleLogs: false, fileOutputMode: {filePath: rawLog, contentType: csv:RAW}};
    io:println(csv:parseString("a,b\n1,x\nz,y\nq,\"r\"", {failSafe: raw}, Rows));
    io:println(check io:fileReadString(rawLog));
    io:println(csv:parseString("a,b\nk,l", {failSafe: raw}, Rows));
    io:println(check io:fileReadString(rawLog));
    raw.fileOutputMode = {filePath: rawLog, contentType: csv:RAW, fileWriteOption: csv:OVERWRITE};
    io:println(csv:parseString("a,b\nm,n", {failSafe: raw}, Rows));
    io:println(check io:fileReadString(rawLog));

    string jsonLog = check file:joinPath(dir, "meta.log");
    csv:FailSafeOptions meta = {enableConsoleLogs: false, fileOutputMode: {filePath: jsonLog, contentType: csv:RAW_AND_METADATA}};
    io:println(csv:parseString("a,b\n1,x\n\"z\",\"y\"", {failSafe: meta}, Rows));
    io:println(csv:transform([{"a": "z", "b": "y"}], {failSafe: meta}, Rows));
    meta.fileOutputMode = {filePath: jsonLog, contentType: csv:METADATA};
    io:println(csv:parseString("a,b\nz,y", {failSafe: meta}, Rows));
    string[] lines = check io:fileReadLines(jsonLog);
    foreach string line in lines {
        int? location = line.indexOf("\"location\"");
        io:println(line.substring(0, 9) == "{\"time\":\"", " ", location is int ? line.substring(location) : line);
    }

    // A log file that cannot be created is an error.
    csv:FailSafeOptions unwritable = {enableConsoleLogs: false, fileOutputMode: {filePath: dir}};
    Rows|csv:Error result = csv:parseString("a,b\nz,y", {failSafe: unwritable});
    io:println(result is csv:Error);
    check file:remove(dir, file:RECURSIVE);
}

// @output [{"a":1,"b":"x"},{"a":2,"b":"w"}]
// @output [{"a":1,"b":"x"},{"a":2,"b":"w"}]
// @output [{"a":1,"b":"x"},{"a":3,"b":"w"}]
// @output true
// @output [{"a":1,"b":"x"}]
// @output [{"a":1,"b":"x"}]
// @output [{"a":1,"b":"x"},{"a":3,"b":"w"}]
// @output error Error ("provided header row is empty")
// @output [{"a":1,"b":"q"}]
// @output [{"a":1,"b":"q"}]
// @output [{"a":1,"b":"x"}]
// @output z,y
// @output q,"r"
// @output []
// @output z,y
// @output q,"r"
// @output k,l
// @output []
// @output m,n
// @output [{"a":1,"b":"x"}]
// @output []
// @output []
// @output true "location":{"row":3,"column":1},"offendingRow":"\"z\",\"y\"","message":"value 'z' cannot be cast into 'int'"}
// @output true "location":{"row":1,"column":1},"offendingRow":"{\"a\":\"z\",\"b\":\"y\"}","message":"no mapping type found for value 'z' in key 'a'"}
// @output true "location":{"row":2,"column":1},"message":"value 'z' cannot be cast into 'int'"}
// @output true
