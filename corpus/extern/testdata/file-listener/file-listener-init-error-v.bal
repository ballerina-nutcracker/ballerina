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

import ballerina/file;
import ballerina/io;

public function testMain() returns error? {
    file:Listener|error emptyPath = new ({path: "", recursive: false});
    if emptyPath is error {
        io:println(emptyPath.message()); // @output 'path' field is empty
    } else {
        io:println("unexpected success");
    }

    file:Listener|error missingDir = new ({path: "testdata/file-listener/no-such-dir", recursive: false});
    if missingDir is error {
        io:println(missingDir.message()); // @output Folder does not exist: testdata/file-listener/no-such-dir
    } else {
        io:println("unexpected success");
    }

    file:Listener|error notADir = new ({path: "testdata/file-listener/fixture.txt", recursive: false});
    if notADir is error {
        io:println(notADir.message()); // @output Unable to find a directory: testdata/file-listener/fixture.txt
    } else {
        io:println("unexpected success");
    }
}
