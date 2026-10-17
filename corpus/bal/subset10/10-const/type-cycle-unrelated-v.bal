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

type Rec [Rec?, C?, D?, E?];

type Other record {| Other? o; |};

type WithDefault record {| int x = 1; |};

const map<int> C = {a: 1};

const map<Other?> D = {b: ()};

const map<WithDefault|int:Signed32> E = {c: {x: 2}, d: 3};

public function main() {
    Rec r = [(), C, D, E];
    io:println(r); // @output [null,{"a":1},{"b":null},{"c":{"x":2},"d":3}]
}
