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
    string blockPath = "/tmp/bal_io_stream_read_blocks1.bin";
    byte[] content = [1, 2, 3, 4, 5, 6, 7];
    check io:fileWriteBytes(blockPath, content);

    stream<io:Block, io:Error?> s = check io:fileReadBlocksAsStream(blockPath, 3);
    record {|io:Block value;|}|io:Error? r1 = s.next();
    if r1 is record {|io:Block value;|} {
        io:println(r1.value.length()); // @output 3
        io:println(r1.value[0]); // @output 1
    }
    record {|io:Block value;|}|io:Error? r2 = s.next();
    if r2 is record {|io:Block value;|} {
        io:println(r2.value.length()); // @output 3
    }
    record {|io:Block value;|}|io:Error? r3 = s.next();
    if r3 is record {|io:Block value;|} {
        io:println(r3.value.length()); // @output 1
        io:println(r3.value[0]); // @output 7
    }
    record {|io:Block value;|}|io:Error? r4 = s.next();
    if r4 is () {
        io:println("done"); // @output done
    }

    string defaultPath = "/tmp/bal_io_stream_read_blocks_default.bin";
    check io:fileWriteBytes(defaultPath, [9, 8, 7]);
    stream<io:Block, io:Error?> defaultStream = check io:fileReadBlocksAsStream(defaultPath);
    record {|io:Block value;|}|io:Error? d1 = defaultStream.next();
    if d1 is record {|io:Block value;|} {
        io:println(d1.value.length()); // @output 3
    }
}
