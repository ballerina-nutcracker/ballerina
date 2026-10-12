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

// A flush names a peer like a send does, so fork workers and the function's
// workers cannot flush each other.
function forkWorkerFlushesFunctionWorker() {
    worker named {
    }
    fork {
        worker a {
            error? _ = flush named; // @error named isn't a peer of a
        }
    }
}

function defaultWorkerFlushesForkWorker() {
    fork {
        worker a {
        }
    }
    error? _ = flush a; // @error a isn't a peer of the default worker
}

function forkWorkerFlushesFunction() {
    fork {
        worker a {
            error? _ = flush function; // @error a fork has no default worker
        }
    }
}
