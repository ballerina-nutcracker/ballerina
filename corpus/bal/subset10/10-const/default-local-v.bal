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

const int STEP1 = 3;

public function main() {
    record {|int z = STEP1;|} folded = {};
    io:println(folded.z); // @output 3

    final int captured = 9;
    record {|int z = captured;|} capturing = {};
    io:println(capturing.z); // @output 9

    record {|int z = STEP1; string s = "a";|} partial = {z: 1};
    io:println(partial); // @output {"z":1,"s":"a"}
}
