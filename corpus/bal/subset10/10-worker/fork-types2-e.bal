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

// These return types fail in type resolution, after symbol resolution has
// accepted the fork, so the statements after the fork still run through it.
function nonConstantLength() {
    int n = 2;
    fork {
        worker w returns int[n] { // @error the length must be a constant
            return [1, 2];
        }
    }
    int[] r = wait w;
    io:println(r);
}

const int N = -1;

function negativeLength() {
    fork {
        worker a returns int {
            return 1;
        }
        worker w returns int[N] { // @error the length must not be negative
            return [];
        }
    }
    int x = wait a;
    io:println(x);
}
