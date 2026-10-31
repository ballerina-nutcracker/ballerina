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

import ballerina/data.jsondata;
import ballerina/io;

type Account record {|
    string id;
    decimal balance;
    float rate;
    boolean active;
    string? note;
    int[] history;
|};

public function main() {
    decimal balance = 10.50;
    Account account = {id: "a1", balance, rate: 0.5, active: true, note: (), history: [1, 2]};
    json j1 = jsondata:toJson(account);
    io:println(j1);
    // @output {"id":"a1","balance":10.50,"rate":0.5,"active":true,"note":null,"history":[1,2]}
    io:println(j1 is map<json>);
    // @output true

    io:println(jsondata:toJson(42));
    // @output 42
    io:println(jsondata:toJson("text"));
    // @output text
    io:println(jsondata:toJson(()) is ());
    // @output true
    io:println(jsondata:toJson(1.25d));
    // @output 1.25
    io:println(jsondata:toJson(xml `<a>b</a>`));
    // @output <a>b</a>

    map<anydata> nested = {list: [xml `<c/>`, {k: [true, ()]}], empty: {}, values: []};
    io:println(jsondata:toJson(nested));
    // @output {"list":["<c/>",{"k":[true,null]}],"empty":{},"values":[]}

    int[] shared = [1, 2];
    anydata[] twice = [shared, shared];
    io:println(jsondata:toJson(twice));
    // @output [[1,2],[1,2]]

    readonly & map<int> frozen = {a: 1};
    json j2 = jsondata:toJson(frozen);
    io:println(j2);
    // @output {"a":1}
    io:println(j2 is readonly);
    // @output false

    [int, string, decimal] tuple = [1, "x", 2.5];
    io:println(jsondata:toJson(tuple));
    // @output [1,"x",2.5]
}
