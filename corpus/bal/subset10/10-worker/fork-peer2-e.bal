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
function workerOfEarlierFork() {
    fork {
        worker a {
        }
    }
    fork {
        worker b {
            int _ = <- a; // @error workers of different forks aren't peers
        }
    }
}

function nestedForkWorkerToOuterSibling() {
    fork {
        worker host {
            fork {
                worker inner {
                    1 -> sib; // @error inner isn't a peer of host's siblings
                }
            }
        }
        worker sib {
        }
    }
}

function forkWorkerToOwnNestedForkWorker() {
    fork {
        worker host {
            fork {
                worker inner {
                }
            }
            1 -> inner; // @error the forking worker isn't a peer of the workers it forks
        }
    }
}

function functionWorkerToOwnForkWorker() {
    worker host {
        fork {
            worker a {
            }
        }
        1 -> a; // @error a function-level worker isn't a peer of the workers it forks
    }
}

function selfSend() {
    fork {
        worker a {
            1 -> a; // @error a worker can't send to itself
        }
        worker b {
        }
    }
}
