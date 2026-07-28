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

type Details record {|
    int arithmetic;
    int bitwise;
    boolean logic;
    string label;
    int[] numbers;
    float converted;
    decimal fraction;
|};

const int BASE = 10;

annotation Details details on type;

@details {
    arithmetic: (BASE + 2) * 3 - 1,
    bitwise: ((~BASE) & 15) << 1,
    logic: !(BASE < 5) && BASE == 10,
    label: string `value-${BASE + 2}`,
    numbers: [BASE, ...[BASE + 1, -3]],
    converted: <float>(BASE + 2),
    fraction: <decimal>BASE / 4
}
type Target record {|
    string id;
|};

public function main() {
    Details? value = Target.@details;
    if value is Details {
        io:println(value.arithmetic); // @output 35
        io:println(value.bitwise); // @output 10
        io:println(value.logic); // @output true
        io:println(value.label); // @output value-12
        io:println(value.numbers[1]); // @output 11
        io:println(value.converted); // @output 12.0
        io:println(value.fraction); // @output 2.5
    }
}
