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

// @labels query-expr from-clause let-clause join-clause
import ballerina/io;

public function main() {
    int[] xs = [1, 2];
    int[] a = from int _ in xs let int _ = 5, string _ = "s" select 7;
    io:println(a); // @output [7,7]
    int[] b = from int x in xs join int _ in [10, 20] on 1 equals 1 select x;
    io:println(b); // @output [1,1,2,2]
    int[] c = from var _ in xs join var _ in [3] on 1 equals 1 let var _ = "q" select 9;
    io:println(c); // @output [9,9]
    int?[] zs = [1, ()];
    int[] d = from int? _ in zs let int? _ = () select 2;
    io:println(d); // @output [2,2]
}
