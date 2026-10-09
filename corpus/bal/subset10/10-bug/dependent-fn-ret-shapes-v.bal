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

function unionRet(int val, typedesc<anydata> T = <>) returns T|error = external;

function arrayRet(typedesc T = <>) returns T[] = external;

function fixedArrayRet(typedesc T = <>) returns T[2] = external;

function arrayInUnionRet(typedesc<anydata> T = <>) returns (T|error)[] = external;

function nilUnionRet(typedesc<int|string> T = <>) returns T? = external;

function intersectionRet(typedesc<anydata> T = <>) returns (T & readonly)|error = external;

annotation Marker on function;

annotation ReturnMarker on return;

@Marker
function annotatedRet(string a, typedesc<anydata> T = <>) returns @ReturnMarker T|error = external;

public function main() {
    io:println("declared"); // @output declared
}
