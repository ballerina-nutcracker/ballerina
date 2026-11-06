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

// A lambda in B's default selects from X|Y while A's defaults are still
// waiting: the trial defers A, and selection resolves A and picks X.
type A record {|
    B? b = ();
    int x = 1;
|};

type B record {|
    isolated function () returns X|Y f = isolated function () returns X|Y {
        return {a: {}};
    };
|};

type X record {|
    A a;
|};

type Y record {|
    A a;
    string s;
|};

public function main() {
    B b = {};
    var g = b.f;
    io:println(g()); // @output {"a":{"b":null,"x":1}}
}
