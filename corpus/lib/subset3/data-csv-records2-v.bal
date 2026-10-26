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

import ballerina/data.csv;
import ballerina/io;

type Required record {|int a; string b;|};

type Nilable record {|int a; string? b;|};

type Optional record {|int a; string b?;|};

type Open record {int a; string b;};

type RequiredRows Required[];

type NilableRows Nilable[];

type OpenRows Open[];

public function main() {
    // A short row cannot leave a required non-nilable field without a value.
    Required[]|csv:Error required = csv:parseString("a,b\n4");
    io:println(required);
    io:println(csv:parseString("a,b\n1,x\n4", {}, OpenRows));
    io:println(csv:parseBytes("a,b\n4".toBytes(), {}, RequiredRows));
    // A nilable field gets `()` and an optional field is left out.
    Nilable[]|csv:Error nilable = csv:parseString("a,b\n4");
    io:println(nilable);
    Optional[]|csv:Error optional = csv:parseString("a,b\n4");
    io:println(optional);
    // With failSafe the short row is logged and skipped.
    io:println(csv:parseString("a,b\n1,x\n4\n3,w", {failSafe: {}}, RequiredRows));
    io:println(csv:parseString("a,b\n1,x\n4\n3,w", {failSafe: {}}, NilableRows));
}

// @output error Error ("no matching header value is found for the required field 'b'")
// @output error Error ("no matching header value is found for the required field 'b'")
// @output error Error ("no matching header value is found for the required field 'b'")
// @output [{"a":4,"b":null}]
// @output [{"a":4}]
// @output [{"a":1,"b":"x"},{"a":3,"b":"w"}]
// @output [{"a":1,"b":"x"},{"a":4,"b":null},{"a":3,"b":"w"}]
