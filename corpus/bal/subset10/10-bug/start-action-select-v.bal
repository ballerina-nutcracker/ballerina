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

function times(int i) returns int {
    return i * 10;
}

public function main() {
    future<int>[] futures = from int i in [1, 2, 3] select start times(i);
    int[] results = [];
    foreach future<int> f in futures {
        int value = checkpanic wait f;
        results.push(value);
    }
    io:println(results); // @output [10,20,30]

    future<int>[] filtered = from int i in [1, 2, 3, 4]
        where i % 2 == 0
        let int j = i + 1
        select start times(j);
    foreach future<int> f in filtered {
        int value = checkpanic wait f;
        io:println(value); // @output 30
                           // @output 50
    }

    future<int>[] casted = from int i in [7] select <future<int>>(start times(i));
    int castedValue = checkpanic wait casted[0];
    io:println(castedValue); // @output 70
}
