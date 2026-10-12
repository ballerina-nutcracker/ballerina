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

// launch returns without joining its fork. The fork's workers keep running and
// keep sharing the captured local after launch has returned: w waits for v to
// update it, and is still observable through the returned future.
function launch(int seed) returns future<int> {
    int captured = seed;
    fork {
        worker w returns int {
            wait v;
            return captured * 2;
        }
        worker v {
            captured += 1;
        }
    }
    return w;
}

public function main() {
    future<int> escaped = launch(5);
    io:println("launched"); // @output launched
    int|error result = wait escaped;
    io:println(result); // @output 12
}
