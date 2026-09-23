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
    map<anydata> m = {"tag": "root"};
    m["self"] = m;
    map<anydata> mClone = m.clone();
    io:println(mClone === m); // @output false
    io:println(mClone["self"] === mClone); // @output true
    io:println(mClone["tag"]); // @output root

    anydata[] arr = [1];
    arr.push(arr);
    anydata[] arrClone = arr.clone();
    io:println(arrClone === arr); // @output false
    io:println(arrClone[1] === arrClone); // @output true

    // A node reachable by two paths is cloned once and stays shared.
    int[] shared = [1, 2];
    anydata[][] dag = [shared, shared];
    anydata[][] dagClone = dag.clone();
    io:println(dagClone[0] === dagClone[1]); // @output true
    io:println(dagClone[0] === shared); // @output false
}
