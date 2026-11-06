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

function inc(int|string arg) returns int {
    final int|string v = arg;
    if v is string {
        return 0;
    }
    // v is final, so the narrowing from default-worker-init holds in the worker.
    worker w returns int {
        return v + 1;
    }
    return wait w;
}

public function main() {
    io:println(inc(4)); // @output 5
    io:println(inc("s")); // @output 0
}
