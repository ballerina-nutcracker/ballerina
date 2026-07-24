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

class LineProvider {
    string[] lines;
    int idx = 0;

    isolated function init(string[] lines) {
        self.lines = lines;
    }

    public isolated function next() returns record {|string value;|}|io:Error? {
        if self.idx >= self.lines.length() {
            return ();
        }
        string line = self.lines[self.idx];
        self.idx += 1;
        return {value: line};
    }
}

public function main() returns error? {
    string path = "/tmp/bal_io_stream_write_lines1.txt";
    stream<string, io:Error?> lineStream = new (new LineProvider(["One", "Two", "Three"]));
    check io:fileWriteLinesFromStream(path, lineStream);

    string[] lines = check io:fileReadLines(path);
    foreach string line in lines {
        io:println(line);
    }

    stream<string, io:Error?> appendStream = new (new LineProvider(["Four"]));
    check io:fileWriteLinesFromStream(path, appendStream, io:APPEND);
    string[] allLines = check io:fileReadLines(path);
    foreach string line in allLines {
        io:println(line);
    }
}
// @output One
// @output Two
// @output Three
// @output One
// @output Two
// @output Three
// @output Four
