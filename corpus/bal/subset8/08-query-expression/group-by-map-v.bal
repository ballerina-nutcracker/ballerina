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
    map<int> xs = {a: 2, b: 1, c: 2, d: 3, e: 1};
    var grouped = from var x in xs
        group by x
        select x;

    var groupedAfterOrder = from var x in xs
        order by x descending
        group by x
        select x;

    io:println(grouped); // @output [2,1,3]
    io:println(groupedAfterOrder); // @output [3,2,1]
}
