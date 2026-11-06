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

function forkWorkerWaitsOnMessagingFunctionWorker() returns int {
    worker named returns int {
        int v = <- function;
        return v * 3;
    }
    5 -> named;
    fork {
        worker f returns int {
            int r = wait named;
            return r + 1;
        }
    }
    return wait f;
}

function functionWorkerWaitsOnMessagingForkWorker() returns int {
    int seed = 4;
    fork {
        worker a {
            seed -> b;
        }
        worker b returns int {
            int v = <- a;
            return v * 2;
        }
    }
    worker named returns int {
        int v = <- function;
        int fromB = wait b;
        return v + fromB;
    }
    seed + 1 -> named;
    return wait named;
}

public function main() {
    io:println(forkWorkerWaitsOnMessagingFunctionWorker()); // @output 16
    io:println(functionWorkerWaitsOnMessagingForkWorker()); // @output 13
}
