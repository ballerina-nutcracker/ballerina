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

type Item record {|
    int id;
|};

public function main() {
    int[] arr = [1, 2];
    [Item[], int]|[string[], int] r = [from int x in arr select {id: x * 3}, 1];
    io:println(r); // @output [[{"id":3},{"id":6}],1]
    io:println(r is [Item[], int]); // @output true
    [[int[]]|[string[]], int]|[[boolean[]]|[string[]], int] o = [[from int x in arr select x + 1], 1];
    io:println(o); // @output [[[2,3]],1]
    io:println(o[0] is [int[]]); // @output true
}
