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

function lambdaInForkWorker() {
    fork {
        worker a {
            var f = function () {
                1 -> b; // @error the lambda runs on its own default worker
            };
            f();
        }
        worker b {
        }
    }
}

function noNamedWorkers() {
    fork {
        worker a {
        }
    }
    1 -> a; // @error the default worker isn't a peer of a fork worker
}

function forkWorkerToFunction() {
    fork {
        worker a {
            1 -> function; // @error a fork has no default worker
        }
        worker b {
            int _ = <- function; // @error a fork has no default worker
        }
    }
}

function forkWorkerToFunctionWorker() {
    worker named {
    }
    fork {
        worker a {
            1 -> named; // @error a function-level worker isn't a peer of a fork worker
        }
        worker b {
            int _ = <- named; // @error a function-level worker isn't a peer of a fork worker
        }
    }
}

function defaultWorkerToForkWorker() {
    worker named {
    }
    fork {
        worker a {
        }
        worker b {
        }
    }
    1 -> a; // @error the default worker isn't a peer of a fork worker
    int _ = <- b; // @error the default worker isn't a peer of a fork worker
}

function namedWorkerToInitForkWorker() {
    fork {
        worker a {
        }
        worker b {
        }
    }
    worker named {
        1 -> a; // @error a function-level worker isn't a peer of a fork worker
        int _ = <- b; // @error a function-level worker isn't a peer of a fork worker
    }
}
