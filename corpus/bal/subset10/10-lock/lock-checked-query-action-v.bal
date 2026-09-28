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

isolated error? status = ();
isolated int total = 0;

function store() returns error? {
    int[] seen = [];
    lock {
        status = check from var value in [1, 2, 3] do {
            seen.push(value);
        };
    }
    lock {
        status = checkpanic from var value in [4, 5] do {
            seen.push(value);
        };
    }
    io:println(seen); // @output [1,2,3,4,5]
}

function accumulate() {
    lock {
        () _ = checkpanic from var value in [1, 2, 3] do {
            total += value;
        };
    }
}

public function main() returns error? {
    check store();
    accumulate();
    lock {
        io:println(status is ()); // @output true
    }
    lock {
        io:println(total); // @output 6
    }
}
