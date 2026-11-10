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

// A fork worker declared in the initialization statements may be evaluated at
// most once (spec 7.3.2), counting references from the function's workers.
function namedWorkerAndDefaultWorker() {
    fork {
        worker f returns int {
            return 1;
        }
    }
    worker named returns int {
        return wait f;
    }
    int a = wait f; // @error f is also referenced by named
    int b = wait named;
    io:println(a + b);
}
