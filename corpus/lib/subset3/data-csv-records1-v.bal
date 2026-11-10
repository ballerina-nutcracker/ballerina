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
    // A short row cannot leave a required non-nilable field without a value.
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
    // A repeated custom or list header keeps the position of its first column and the value of its last.
    io:println(csv:parseString("1,2,3,4,5", {header: (), customHeadersIfHeadersAbsent: ["b", "a", "c", "b", "a"]}, OpenRows));
    io:println(csv:parseList([["b", "a", "c", "b", "a"], ["1", "2", "3", "4", "5"]], {headerRows: 1}, OpenRows));
    string[] headers = ["a"];
    string[] row = ["0"];
    string line = "0";
    foreach int i in 0 ..< 40 {
        headers.push("c" + i.toString());
        row.push((i + 1).toString());
        line += "," + (i + 1).toString();
    }
    headers.push("c3", "a", "c0");
    row.push("41", "42", "43");
    line += ",41,42,43";
    io:println(csv:parseString(line, {header: (), customHeadersIfHeadersAbsent: headers}, OpenRows));
    io:println(csv:parseList([headers, row], {headerRows: 1}, OpenRows));
}

// @output [{"a":1,"b":"x"}]
// @output error Error ("no mapping field in the expected type for header 'c'")
// @output [{"a":1,"b":"x","c":9}]
// @output [{"a":1,"b":2}]
// @output error Error ("no matching header value is found for the required field 'b'")
// @output error Error ("no matching header value is found for the required field 'a'")
// @output error Error ("no matching header value is found for the required field 'b'")
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
// @output [{"a":5,"b":4,"c":3}]
// @output [{"a":5,"b":4,"c":3}]
// @output [{"a":42,"c0":43,"c1":2,"c2":3,"c3":41,"c4":5,"c5":6,"c6":7,"c7":8,"c8":9,"c9":10,"c10":11,"c11":12,"c12":13,"c13":14,"c14":15,"c15":16,"c16":17,"c17":18,"c18":19,"c19":20,"c20":21,"c21":22,"c22":23,"c23":24,"c24":25,"c25":26,"c26":27,"c27":28,"c28":29,"c29":30,"c30":31,"c31":32,"c32":33,"c33":34,"c34":35,"c35":36,"c36":37,"c37":38,"c38":39,"c39":40}]
// @output [{"a":42,"c0":43,"c1":2,"c2":3,"c3":41,"c4":5,"c5":6,"c6":7,"c7":8,"c8":9,"c9":10,"c10":11,"c11":12,"c12":13,"c13":14,"c14":15,"c15":16,"c16":17,"c17":18,"c18":19,"c19":20,"c20":21,"c21":22,"c22":23,"c23":24,"c24":25,"c25":26,"c26":27,"c27":28,"c28":29,"c29":30,"c30":31,"c31":32,"c32":33,"c33":34,"c34":35,"c35":36,"c36":37,"c37":38,"c38":39,"c39":40}]
