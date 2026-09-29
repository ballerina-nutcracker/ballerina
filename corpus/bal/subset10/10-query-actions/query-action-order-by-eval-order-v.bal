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

int[] log = [];

function t(int v) returns int {
    log.push(v);
    return v;
}

public function main() {
    // The where expression and the order by key of one frame are evaluated
    // before the next frame is pulled; the do body runs once the frames are sorted.
    from var x in [3, 1, 2]
        where t(x * 10) > 0
        order by t(x) descending
        do {
            log.push(x + 1000);
        };
    io:println(log); // @output [30,3,10,1,20,2,1003,1002,1001]

    log = [];
    int[] sorted = from var x in [3, 1, 2]
        from var y in [t(x * 100)]
        where t(x * 10) > 0
        order by t(x) descending
        select x + y;
    io:println(log); // @output [300,30,3,100,10,1,200,20,2]
    io:println(sorted); // @output [303,202,101]

    log = [];
    from var x in [2, 1, 3]
        let int key = t(x)
        order by key
        limit 2
        order by key descending
        do {
            log.push(x + 100);
        };
    io:println(log); // @output [2,1,3,102,101]
}
