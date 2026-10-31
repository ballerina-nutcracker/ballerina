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

type Employee record {|
    int id;
    string name;
    decimal? salary;
|};

public function main() returns error? {
    string csvString = string `id,name,salary
1,"Doe, John",100.50
# a comment line
2,Jane,null`;

    // Parse into records and into tuples.
    Employee[] employees = check csv:parseString(csvString);
    io:println(employees);
    [int, string][] pairs = check csv:parseString(csvString, {skipLines: [1]});
    io:println(pairs);

    // Keep the header row when the result is a list of lists.
    string[][] rows = check csv:parseString(csvString, {outputWithHeaders: true});
    io:println(rows);

    // Parse bytes and a byte-block stream.
    byte[] bytes = "id,name,salary\n3,Max,9".toBytes();
    Employee[] fromBytes = check csv:parseBytes(bytes);
    io:println(fromBytes);
    byte[][] blocks = ["id,name,sal".toBytes(), "ary\n4,Ann,7.25".toBytes()];
    stream<byte[], error?> byteStream = blocks.toStream();
    Employee[] fromStream = check csv:parseStream(byteStream);
    io:println(fromStream);

    // Convert records to lists, choosing the column order.
    anydata[][] columns = check csv:transform(employees, {headerOrder: ["name", "id", "salary"]});
    io:println(columns);

    // Bind string lists to records using custom headers.
    record {|int id; string name;|}[] people = check csv:parseList([["5", "Kim"]], {customHeaders: ["id", "name"]});
    io:println(people);

    // Errors are returned as csv:Error values.
    Employee[]|csv:Error bad = csv:parseString("id,name,salary\nx,Bob,1");
    io:println(bad);
}

// @output [{"id":1,"name":"Doe, John","salary":100.5},{"id":2,"name":"Jane","salary":null}]
// @output [[2,"Jane"]]
// @output [["id","name","salary"],["1","Doe, John","100.50"],["2","Jane","null"]]
// @output [{"id":3,"name":"Max","salary":9.0}]
// @output [{"id":4,"name":"Ann","salary":7.25}]
// @output [["Doe, John",1,100.5],["Jane",2,null]]
// @output [{"id":5,"name":"Kim"}]
// @output error Error ("value 'x' cannot be cast into 'int'")
