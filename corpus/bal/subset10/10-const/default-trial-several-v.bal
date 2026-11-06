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

type A1 record {|
    A2? a2 = ();
    int p = 1;
|};

type A2 record {|
    B? b = ();
    int q = 2;
|};

type B record {|
    X|Y f = {a1: {}, a2: {}};
|};

type X record {|
    A1 a1;
    A2 a2;
|};

type Y record {|
    A1 a1;
    A2 a2;
    string s;
|};

public function main() {
    B b = {};
    io:println(b.f); // @output {"a1":{"a2":null,"p":1},"a2":{"b":null,"q":2}}
}
