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

// othersWritableDir: r-xrwxrwx, owned by the test process. Group and others
// may write to it, but its owner may not.
isolated function othersWritableDir() returns string = external;

public function testMain() returns error? {
    // directory writability reflects the caller's access, not any write bit
    string othersWritable = othersWritableDir();
    io:println(check file:test(othersWritable, file:WRITABLE)); // @output false
    file:MetaData othersWritableMeta = check file:getMetaData(othersWritable);
    io:println(othersWritableMeta.writable); // @output false
}
