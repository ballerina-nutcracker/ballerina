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

type Point record {|
    int x;
    int y;
|};

public function main() {
    map<Point> holder = {a: {x: 9, y: 10}};
    // The elvis operand's type is the union of the member type and the nested constructor's
    // own inferred type, which are separate mapping atoms even though they are structurally
    // the same.
    var fromElvis = {...holder["missing"] ?: {x: 0, y: 0}, z: 5}; // @error future: spread of a non-atomic mapping type
    io:println(fromElvis); // @output {"x":0,"y":0,"z":5}
}
