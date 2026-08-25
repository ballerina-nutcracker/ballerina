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

public function main() returns error? {
    string path = "/tmp/bal_io_stream_read_lines2.txt";
    // Raw bytes for "a\r\nb\rc": a CRLF terminator, then a lone CR terminator,
    // then final content with no trailing terminator (EOF-terminated).
    check io:fileWriteBytes(path, [97, 13, 10, 98, 13, 99]);

    stream<string, io:Error?> s = check io:fileReadLinesAsStream(path);
    record {|string value;|}|io:Error? r1 = s.next();
    if r1 is record {|string value;|} {
        io:println(r1.value); // @output a
    }
    record {|string value;|}|io:Error? r2 = s.next();
    if r2 is record {|string value;|} {
        io:println(r2.value); // @output b
    }
    record {|string value;|}|io:Error? r3 = s.next();
    if r3 is record {|string value;|} {
        io:println(r3.value); // @output c
    }
    record {|string value;|}|io:Error? r4 = s.next();
    if r4 is () {
        io:println("done"); // @output done
    }
}
