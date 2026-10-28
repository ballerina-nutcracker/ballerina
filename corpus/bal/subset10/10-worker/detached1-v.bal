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

// launch returns without joining its worker. The worker keeps the captured
// frame alive and is still observable through the returned future.
function launch() returns future<int> {
    int captured = 5;
    worker w returns int {
        return captured * 2;
    }
    return w;
}

public function main() {
    future<int> escaped = launch();
    int|error result = wait escaped;
    io:println(result); // @output 10
}
