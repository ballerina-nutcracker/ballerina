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

// Every execution of a fork has its own workers, so one reference after a
// fork in a loop body is evaluated once per worker.
function refInLoop() {
    int sum = 0;
    foreach int i in 1 ... 3 {
        fork {
            worker w returns int {
                return i;
            }
        }
        sum += wait w;
    }
    io:println(sum); // @output 6
}

function waitsOnFunctionWorker() {
    worker named returns int {
        return 5;
    }
    fork {
        worker f returns int {
            int x = wait named;
            return x * 2;
        }
    }
    int r = wait f;
    io:println(r); // @output 10
}

public function main() {
    refInLoop();
    waitsOnFunctionWorker();
}
