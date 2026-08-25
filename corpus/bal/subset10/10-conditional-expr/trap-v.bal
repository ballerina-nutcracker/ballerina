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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

function panicValue() returns int { panic error("boom"); }

public function main() {
    int|error selected = trap (true ? panicValue() : 1);
    int unselected = false ? panicValue() : 2;
    int|error trueArmTrap = true ? trap panicValue() : 3;
    int|error falseArmTrap = false ? 4 : trap panicValue();
    io:println(selected is error); // @output true
    io:println(unselected); // @output 2
    io:println(trueArmTrap is error); // @output true
    io:println(falseArmTrap is error); // @output true
}
