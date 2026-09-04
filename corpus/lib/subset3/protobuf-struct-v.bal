// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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
import ballerina/protobuf.types.'struct as strct;
import ballerina/protobuf.types.'any as pbany;

public function main() returns error? {
    map<anydata> flat = {"a": 1, "b": "two", "c": true};
    strct:ContextStruct ctxStruct = {content: flat, headers: {}};
    io:println(ctxStruct.content["b"]); // @output two

    pbany:Any packedFlat = check pbany:pack(flat);
    io:println(packedFlat.typeUrl); // @output type.googleapis.com/google.protobuf.Struct
    map<anydata> unpackedFlat = check pbany:unpack(packedFlat);
    io:println(unpackedFlat["a"]); // @output 1.0
    io:println(unpackedFlat["b"]); // @output two
    io:println(unpackedFlat["c"]); // @output true

    // Nested map and list values, and a nil field.
    map<anydata> nested = {
        "inner": {"x": 1, "y": 2},
        "list": [1, "two", true, ()],
        "empty": ()
    };
    pbany:Any packedNested = check pbany:pack(nested);
    map<anydata> unpackedNested = check pbany:unpack(packedNested);
    anydata inner = unpackedNested["inner"];
    if inner is map<anydata> {
        io:println(inner["x"]); // @output 1.0
    }
    anydata list = unpackedNested["list"];
    if list is anydata[] {
        io:println(list.length()); // @output 4
        io:println(list[1]); // @output two
    }
    io:println(unpackedNested["empty"]); // @output

    // decimal values widen to float64, like int, since Struct only carries doubles.
    map<anydata> withDecimal = {"d": 3.14d, "inner": {"x": 9.9d}, "list": [1.5d, 2]};
    pbany:Any packedDecimal = check pbany:pack(withDecimal);
    map<anydata> unpackedDecimal = check pbany:unpack(packedDecimal);
    io:println(unpackedDecimal["d"]); // @output 3.14
    anydata decimalInner = unpackedDecimal["inner"];
    if decimalInner is map<anydata> {
        io:println(decimalInner["x"]); // @output 9.9
    }
    anydata decimalList = unpackedDecimal["list"];
    if decimalList is anydata[] {
        io:println(decimalList[0]); // @output 1.5
        io:println(decimalList[1]); // @output 2.0
    }
}
