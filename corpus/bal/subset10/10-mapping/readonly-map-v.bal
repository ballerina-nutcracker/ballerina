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

public function main() {
    map<int> m = {readonly frozen: 1, mutable: 2};
    m["mutable"] = 20;
    m["added"] = 3;
    io:println(m); // @output {"frozen":1,"mutable":20,"added":3}
    io:println(m["frozen"]); // @output 1

    map<int> marked = {readonly a: 1};
    map<int> unmarked = {a: 1};
    unmarked["a"] = 2;
    io:println(marked); // @output {"a":1}
    io:println(unmarked); // @output {"a":2}
}
