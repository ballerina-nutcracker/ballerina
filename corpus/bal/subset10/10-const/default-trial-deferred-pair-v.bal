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

// Both candidates of X|Y hold a record whose defaults are still waiting, so
// only those defaults decide which candidate fits each constructor.
type A record {|
    C? c = ();
    int x = 1;
|};

type C record {|
    B? b = ();
    string s;
|};

type B record {|
    X|Y f = {a: {}};
    X|Y g = {a: {s: "k"}};
|};

type X record {|
    A a;
|};

type Y record {|
    C a;
|};

function isX(any v) returns boolean {
    return v is X;
}

public function main() {
    B b = {};
    io:println(b.f); // @output {"a":{"c":null,"x":1}}
    io:println(isX(b.f)); // @output true
    io:println(b.g); // @output {"a":{"s":"k","b":null}}
    io:println(isX(b.g)); // @output false
}
