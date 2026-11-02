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
    int[] xs = [1, 2, 3];
    int n = from var x in xs
        let int y = x * 2
        collect [y].length();
    io:println(n); // @output 3

    int[] r = from var x in xs
        let int m = x + 10
        group by x
        select [m][0] + [m].length();
    io:println(r); // @output [12,13,14]

    int[][] withOtherMembers = from var x in [1, 2, 1]
        let int m = x + 10
        group by x
        select [0, m, 1];
    io:println(withOtherMembers); // @output [[0,11,11,1],[0,12,1]]
}
