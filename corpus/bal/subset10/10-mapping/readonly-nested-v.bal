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
    map<map<int>> nestedMapping = {readonly inner: {x: 1}, other: {y: 2}};
    io:println(nestedMapping); // @output {"inner":{"x":1},"other":{"y":2}}
    nestedMapping["other"] = {y: 20};
    io:println(nestedMapping["other"]); // @output {"y":20}

    map<int[]> nestedList = {readonly inner: [1, 2]};
    io:println(nestedList); // @output {"inner":[1,2]}

    int[] & readonly frozen = [3, 4];
    map<int[]> referenced = {readonly inner: frozen};
    io:println(referenced); // @output {"inner":[3,4]}
}
