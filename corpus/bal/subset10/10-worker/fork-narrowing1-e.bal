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

function reassignedByForkWorker(int|string arg) returns int {
    int|string v = arg;
    if v is int {
        fork {
            worker w {
                v = "no longer an int";
            }
        }
        wait w;
        return v; // @error v is captured by a fork worker
    }
    return 0;
}

function onlyReadByOneWorker(int|string arg) returns int {
    int|string v = arg;
    fork {
        worker reader {
            io:println(v);
        }
        worker writer {
            v = "changed";
        }
    }
    if v is int {
        return v; // @error v is captured by the fork workers
    }
    return 0;
}

function narrowedInSiblingBody(int|string arg) returns int {
    int|string v = arg;
    fork {
        worker a {
            v = "no longer an int";
        }
        worker b returns int {
            if v is int {
                return v; // @error v is captured by worker a
            }
            return 0;
        }
    }
    return wait b;
}
