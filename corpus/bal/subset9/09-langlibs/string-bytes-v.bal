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

import ballerina/io;

public function main() {
    byte[] bytes = "hi".toBytes();
    io:println(bytes.length()); // @output 2

    string|error back = string:fromBytes(bytes);
    io:println(back); // @output hi

    byte[] invalid = [255, 254];
    string|error invalidStr = string:fromBytes(invalid);
    io:println(invalidStr is error); // @output true
}
