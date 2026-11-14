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

type Meta record {|
    string name;
|};

annotation Meta ParamInfo on parameter;

annotation Meta ReturnInfo on return;

public function main() {
    var wrongValueType = function(@ParamInfo {name: 1} int x) returns int => x; // @error
    io:println(wrongValueType(1));

    var missingValue = function(@ParamInfo int x) returns int => x; // @error
    io:println(missingValue(1));

    var wrongReturnValueType = function() returns @ReturnInfo {name: 1} int => 1; // @error
    io:println(wrongReturnValueType());
}
