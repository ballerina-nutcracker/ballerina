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

type R record {|
    int y;
    int x;
    int...;
|};

public function main() {
    R r = {"b": 3, "a": 4, y: 2, x: 1};
    // The spec doesn't require a particular output order here,
    // but we put required fields in sorted order,
    // followed by optional fields in order specified in constructor
    io:println(r); // @output {"b":3,"a":4,"y":2,"x":1}
}
