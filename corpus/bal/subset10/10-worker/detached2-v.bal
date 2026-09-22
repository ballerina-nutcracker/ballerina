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

// A returned worker future is an ordinary future once it leaves the declaring
// function, so the single-reference rule no longer applies to it and repeated
// waits fall back to the existing claim behaviour: the first wait yields the
// value and every later one yields an error.
function launch() returns future<int> {
    worker w returns int {
        return 7;
    }
    return w;
}

public function main() {
    future<int> escaped = launch();
    int|error first = wait escaped;
    int|error second = wait escaped;
    io:println(first); // @output 7
    io:println(second); // @output error("multiple waits on the same future is not allowed")
}
