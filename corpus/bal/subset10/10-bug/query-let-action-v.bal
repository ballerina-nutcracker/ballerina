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

function one() returns int => 1;

client class Doubler {
    remote function apply(int i) returns int => i * 2;
}

public function main() {
    int[] r = from int _ in [1] let future<int> f = start one() select checkpanic wait f;
    io:println(r); // @output [1]

    int[] braced = from int _ in [1] let future<int> f = (start one()) select checkpanic wait f;
    io:println(braced); // @output [1]

    int[] inferred = from int _ in [1] let var f = start one() select checkpanic wait f;
    io:println(inferred); // @output [1]

    Doubler d = new;
    int[] doubled = from int i in [1, 2] let int x = d->apply(i) select x;
    io:println(doubled); // @output [2,4]

    int[] waited = from int i in [1]
        let future<int> f = start one()
        let int|error x = wait f
        select checkpanic x + i;
    io:println(waited); // @output [2]

    int[] ordered = from int i in [3, 1] let future<int> f = start one() let int w = checkpanic wait f order by i select i + w;
    io:println(ordered); // @output [2,4]

    int[] limited = from int _ in [3, 1] limit 1 let future<int> f = start one() select checkpanic wait f;
    io:println(limited); // @output [1]
}
