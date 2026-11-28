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

function isFuture(future<boolean> f) returns boolean {
    return f is future<boolean>;
}

public function main() {
    // A fork worker's name is not in scope in its own body.
    fork {
        worker w returns boolean {
            return isFuture(w); // @error w is not in scope in its own body
        }
    }
    io:println("started");
}

function nestedFork() {
    // Nor in a fork nested in its own body.
    fork {
        worker host returns boolean {
            fork {
                worker inner returns boolean {
                    return isFuture(host); // @error host is not in scope in its own body
                }
            }
            return wait inner;
        }
    }
}
