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
import ballerina/lang.value;

type Person record {|
    string name;
|};

isolated class Taker {
    final string got;

    isolated function init(Person|error person) {
        self.got = person is error ? person.message() : person.name;
    }
}

type A record {|
    Person|error p;
|};

type B record {|
    Person|error p;
    int n;
|};

final boolean flag = false;
final json & readonly ro = {name: "ann"};

type Holder record {|
    Taker t = new (flag ? convert(ro) : value:cloneWithType(ro));
|};

public function main() {
    json j = {name: "ann"};
    Taker a = new (value:cloneWithType(j));
    io:println(a.got); // @output ann
    Taker b = new (j.cloneWithType());
    io:println(b.got); // @output ann
    Holder h = {};
    io:println(h.t.got); // @output ann
    A|B m = {p: value:cloneWithType(j)};
    Person|error p = m.p;
    io:println(p is error ? p.message() : p.name); // @output ann
}

isolated function convert(json v, typedesc<anydata> td = <>) returns td|error = external;
