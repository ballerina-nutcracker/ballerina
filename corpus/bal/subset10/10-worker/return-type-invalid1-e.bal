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

const int NEGATIVE = -1;

function nonConstantLength() {
    int n = 2;
    worker w returns int[n] { // @error array length must be a constant
        return [1, 2];
    }
    var result = wait w;
    io:println(result);
}

function negativeLength() {
    worker w returns int[NEGATIVE] { // @error array length must not be negative
        return [];
    }
    var result = wait w;
    io:println(result);
}

function emptyIntersection() {
    worker w returns int[] & map<int> { // @error intersection is empty
        return [];
    }
    var result = wait w;
    io:println(result);
}

public function main() {
    nonConstantLength();
    negativeLength();
    emptyIntersection();
}
