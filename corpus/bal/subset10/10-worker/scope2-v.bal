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

// Each worker body has its own scope, so sibling workers may reuse a name for
// unrelated locals.
public function main() {
    worker first returns int {
        int shared = 1;
        return shared;
    }
    worker second returns string {
        string shared = "two";
        return shared;
    }
    int firstResult = wait first;
    string secondResult = wait second;
    io:println(firstResult); // @output 1
    io:println(secondResult); // @output two
}
