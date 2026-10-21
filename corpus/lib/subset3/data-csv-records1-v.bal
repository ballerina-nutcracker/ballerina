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

type Closed record {|int a; string b;|};

type Open record {int a;};

type Nilable record {|int a; int? b; string c?;|};

type Rest record {|int a; int...;|};

type Positional record {|int '1; string '2;|};

type ClosedRows Closed[];

type OpenRows Open[];

type NilableRows Nilable[];

type RestRows Rest[];

type PositionalRows Positional[];

type ClosedPair Closed[2];

type AnyRecords record {}[2];

type ReadonlyRows readonly & Closed[];

public function main() {
    // Unknown columns are dropped from a closed record unless projection is disabled.
    io:println(csv:parseString("a,b,c\n1,x,9", {}, ClosedRows));
    io:println(csv:parseString("a,b,c\n1,x,9", {allowDataProjection: false}, ClosedRows));
    // Open records keep unknown columns as rest fields.
    io:println(csv:parseString("a,b,c\n1,x,9", {}, OpenRows));
    // Values that do not convert to the rest field type are dropped.
    io:println(csv:parseString("a,b,c\n1,2,zz", {}, RestRows));
    // A required field without a matching header is an error.
    io:println(csv:parseString("a\n1", {}, ClosedRows));
    io:println(csv:parseString("b\n1", {}, OpenRows));
    // A short row leaves the missing field present with ().
    io:println(csv:parseString("a,b\n1", {}, ClosedRows));
    io:println(csv:parseString("a,b\nz,y", {}, ClosedRows));
    io:println(csv:parseString("a,b\n1,null\n2,()\n3,NULL", {}, NilableRows));
    // nilAsOptionalField leaves an optional field absent for a nil value.
    io:println(csv:parseString("a,b,c\n1,2,null", {allowDataProjection: {nilAsOptionalField: true}}, NilableRows));
    io:println(csv:parseString("a,b,c\n1,2,null", {}, NilableRows));
    // absentAsNilableType allows a nilable field without a header.
    io:println(csv:parseString("a,c\n1,x", {allowDataProjection: {absentAsNilableType: true}}, NilableRows));
    io:println(csv:parseString("a,c\n1,x", {}, NilableRows));
    io:println(csv:parseString("a,b\n1,N/A\n2,null", {nilValue: "N/A"}, NilableRows));
    // Without a header row, columns are named "1", "2", ...
    io:println(csv:parseString("1,x\n2,y", {header: ()}, PositionalRows));
    // More columns than custom headers is an error for record targets.
    io:println(csv:parseString("1,x,3", {header: (), customHeadersIfHeadersAbsent: ["a", "b"]}, ClosedRows));
    // A closed result array keeps its first rows and pads a short result.
    io:println(csv:parseString("a,b\n1,x\n2,y\n3,z", {}, ClosedPair));
    io:println(csv:parseString("a\n1", {}, AnyRecords));
    io:println(csv:parseString("a,b\n1,x", {}, ReadonlyRows));
    // Duplicate headers are an error.
    io:println(csv:parseString("a,a\n1,2", {}, OpenRows));
}

// @output [{"a":1,"b":"x"}]
// @output error Error ("no mapping field in the expected type for header 'c'")
// @output [{"a":1,"b":"x","c":9}]
// @output [{"a":1,"b":2}]
// @output error Error ("no matching header value is found for the required field 'b'")
// @output error Error ("no matching header value is found for the required field 'a'")
// @output [{"a":1,"b":null}]
// @output error Error ("value 'z' cannot be cast into 'int'")
// @output [{"a":1,"b":null},{"a":2,"b":null},{"a":3,"b":null}]
// @output [{"a":1,"b":2}]
// @output [{"a":1,"b":2,"c":"null"}]
// @output [{"a":1,"c":"x","b":null}]
// @output error Error ("no matching header value is found for the required field 'b'")
// @output error Error ("value 'null' cannot be cast into 'int?'")
// @output [{"1":1,"2":"x"},{"1":2,"2":"y"}]
// @output error Error ("invalid number of headers")
// @output [{"a":1,"b":"x"},{"a":2,"b":"y"}]
// @output [{"a":1},{}]
// @output [{"a":1,"b":"x"}]
// @output error Error ("duplicate header found: 'a'")
