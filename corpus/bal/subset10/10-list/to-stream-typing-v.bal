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
import ballerina/lang.array;

type Item record {|
    int id;
|};

function allListUnion(int[]|string[] values) returns stream<int|string, ()> {
    return array:toStream(values);
}

public function main() {
    int[] values = [1, 2];
    stream<int, ()> positional = array:toStream(values);
    stream<int, ()> named = array:toStream(arr = values);
    stream<int, ()> method = values.toStream();
    stream<int, ()> repeated = values.toStream();

    int[2] fixed = [3, 4];
    stream<int, ()> fixedStream = fixed.toStream();
    [int, string] tuple = [5, "five"];
    stream<int|string, ()> tupleStream = tuple.toStream();

    readonly & int[] readonlyArray = [6, 7];
    stream<int, ()> readonlyArrayStream = readonlyArray.toStream();
    readonly & [int, string] readonlyTuple = [8, "eight"];
    stream<int|string, ()> readonlyTupleStream = readonlyTuple.toStream();

    stream<int|string, ()> unionStream = allListUnion([9, 10]);
    Item[] records = [{id: 11}];
    stream<Item, ()> recordStream = records.toStream();
    (int|string)[] unionMembers = [12, "twelve"];
    stream<int|string, ()> unionMemberStream = unionMembers.toStream();

    io:println(positional.next() is record {| int value; |}); // @output true
    io:println(named.next() is record {| int value; |}); // @output true
    io:println(method.next() is record {| int value; |}); // @output true
    io:println(repeated.next() is record {| int value; |}); // @output true
    io:println(fixedStream.next() is record {| int value; |}); // @output true
    io:println(tupleStream.next() is record {| int|string value; |}); // @output true
    io:println(readonlyArrayStream.next() is record {| int value; |}); // @output true
    io:println(readonlyTupleStream.next() is record {| int|string value; |}); // @output true
    io:println(unionStream.next() is record {| int|string value; |}); // @output true
    io:println(recordStream.next() is record {| Item value; |}); // @output true
    io:println(unionMemberStream.next() is record {| int|string value; |}); // @output true
}
