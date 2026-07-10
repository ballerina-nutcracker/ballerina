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

type MediumInfo record {|
    int id;
    int[64] values;
|};

annotation MediumInfo medium on type;

@medium {id: 1, values: []}
type T1 int;
@medium {id: 2, values: []}
type T2 int;
@medium {id: 3, values: []}
type T3 int;
@medium {id: 4, values: []}
type T4 int;
@medium {id: 5, values: []}
type T5 int;
@medium {id: 6, values: []}
type T6 int;
@medium {id: 7, values: []}
type T7 int;
@medium {id: 8, values: []}
type T8 int;

public function main() {
    MediumInfo? first = T1.@medium;
    MediumInfo? last = T8.@medium;
    if first is MediumInfo && last is MediumInfo {
        io:println(first.id); // @output 1
        io:println(last.id); // @output 8
        io:println(last.values[63]); // @output 0
    }
}
