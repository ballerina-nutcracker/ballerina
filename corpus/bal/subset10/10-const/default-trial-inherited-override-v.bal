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

// P overrides the inherited x without a default, so P does not fit {y: 2}
// while its defaults are resolving and Q is the only applicable member.
type Base record {|
    int x = 1;
|};

type P record {|
    *Base;
    int x;
    int y;
    (P|Q)? next = {y: 2};
|};

type Q record {|
    int y;
|};

function isQ(any v) returns boolean {
    return v is Q;
}

public function main() {
    P p = {x: 5, y: 0};
    io:println(p); // @output {"x":5,"y":0,"next":{"y":2}}
    io:println(isQ(p.next)); // @output true
}
