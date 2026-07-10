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

const int BASE = 100;

type HeavyInfo record {|
    int id;
    int[1024] values;
    string label?;
    int[] prefix?;
|};

annotation HeavyInfo heavy on type;

@heavy {
    id: <int>(+(BASE + 1)),
    values: [],
    label: string `heavy-${BASE + 1}`,
    prefix: [BASE, ...[BASE + 1]]
}
type T1 int;
@heavy {id: BASE + 2, values: []}
type T2 int;
@heavy {id: BASE + 3, values: []}
type T3 int;
@heavy {id: BASE + 4, values: []}
type T4 int;
@heavy {id: BASE + 5, values: []}
type T5 int;
@heavy {id: BASE + 6, values: []}
type T6 int;
@heavy {id: BASE + 7, values: []}
type T7 int;
@heavy {id: BASE + 8, values: []}
type T8 int;

public function main() {
    HeavyInfo? first = T1.@heavy;
    HeavyInfo? last = T8.@heavy;
    if first is HeavyInfo && last is HeavyInfo {
        io:println(first.id); // @output 101
        io:println(last.id); // @output 108
        io:println(last.values[1023]); // @output 0
    }
}
