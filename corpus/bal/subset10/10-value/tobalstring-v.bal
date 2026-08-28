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

type Address record {|
    string city;
    string country;
|};

public function main() {
    int i = 12;
    io:println(i.toBalString()); // @output 12

    string s = "Anne";
    io:println(s.toBalString()); // @output "Anne"

    boolean b = true;
    io:println(b.toBalString()); // @output true

    float f1 = 12.342;
    io:println(f1.toBalString()); // @output 12.342

    float nanVal = 0.0 / 0.0;
    io:println(nanVal.toBalString()); // @output float:NaN

    float infVal = 1.0 / 0.0;
    io:println(infVal.toBalString()); // @output float:Infinity

    decimal d1 = 345.2425341;
    io:println(d1.toBalString()); // @output 345.2425341d

    decimal d2 = 1;
    io:println(d2.toBalString()); // @output 1d

    () nilVal = ();
    io:println(nilVal.toBalString()); // @output ()

    anydata[] data = [1, "Sam", 12.3, 12.12d, {value: 12}];
    io:println(data.toBalString()); // @output [1,"Sam",12.3,12.12d,{"value":12}]

    record {} key = {id: 5};
    io:println(key.toBalString()); // @output {"id":5}

    Address addr = {city: "Colombo", country: "Sri Lanka"};
    io:println(addr.toBalString()); // @output {"city":"Colombo","country":"Sri Lanka"}

    map<anydata> nested = {"a": 1, "b": {"c": 2}};
    io:println(nested.toBalString()); // @output {"a":1,"b":{"c":2}}

    int[] nums = [1, 2, 3];
    io:println(nums.toBalString()); // @output [1,2,3]
}
