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

const int BASE = 10;

type TinyInfo record {|
    int value;
|};

annotation TinyInfo tiny on type;

@tiny {value: BASE + 1}
type First int;

@tiny {value: BASE + 2}
type Second int;

public function main() {
    TinyInfo? first = First.@tiny;
    TinyInfo? second = Second.@tiny;
    if first is TinyInfo && second is TinyInfo {
        io:println(first.value); // @output 11
        io:println(second.value); // @output 12
    }
}
