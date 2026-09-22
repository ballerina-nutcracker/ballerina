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

type MR record {|
    readonly int value;
    int otherValue;
|};

type M record {|
    int value;
    int otherValue;
|};

type Opt record {|
    readonly int value?;
    int otherValue;
|};

type RoMap readonly & map<int>;

public function main() {
    MR mr = {value: 5, otherValue: 6};
    MR & readonly ro = {value: 7, otherValue: 8};
    M m = mr;
    map<int> mp = mr;
    MR|M u = mr;
    Opt opt = {value: 11, otherValue: 12};
    RoMap rm = {"a": 9};

    io:println(mr.value); // @output 5
    io:println(mr["value"]); // @output 5
    io:println(ro.value); // @output 7
    io:println(ro["value"]); // @output 7
    io:println(m.value); // @output 5
    io:println(mp["value"]); // @output 5
    io:println(u.value); // @output 5
    io:println(opt?.value); // @output 11
    io:println(mr.value + 1); // @output 6
    io:println(rm["a"]); // @output 9
    mr.otherValue += mr.value;
    io:println(mr.otherValue); // @output 11
}
