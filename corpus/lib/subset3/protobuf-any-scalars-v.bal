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
import ballerina/protobuf.types.'any as pbany;

public function main() returns error? {
    pbany:Any packedInt = check pbany:pack(42);
    io:println(packedInt.typeUrl); // @output type.googleapis.com/google.protobuf.Int64Value
    int unpackedInt = check pbany:unpack(packedInt);
    io:println(unpackedInt); // @output 42

    pbany:Any packedNegInt = check pbany:pack(-7);
    int unpackedNegInt = check pbany:unpack(packedNegInt);
    io:println(unpackedNegInt); // @output -7

    pbany:Any packedFloat = check pbany:pack(1.5);
    io:println(packedFloat.typeUrl); // @output type.googleapis.com/google.protobuf.FloatValue
    float unpackedFloat = check pbany:unpack(packedFloat);
    io:println(unpackedFloat); // @output 1.5

    pbany:Any packedStr = check pbany:pack("hello world");
    io:println(packedStr.typeUrl); // @output type.googleapis.com/google.protobuf.StringValue
    string unpackedStr = check pbany:unpack(packedStr);
    io:println(unpackedStr); // @output hello world

    pbany:Any packedBool = check pbany:pack(false);
    io:println(packedBool.typeUrl); // @output type.googleapis.com/google.protobuf.BoolValue
    boolean unpackedBool = check pbany:unpack(packedBool);
    io:println(unpackedBool); // @output false

    byte[] bytes = [10, 20, 30, 255];
    pbany:Any packedBytes = check pbany:pack(bytes);
    io:println(packedBytes.typeUrl); // @output type.googleapis.com/google.protobuf.BytesValue
    byte[] unpackedBytes = check pbany:unpack(packedBytes);
    io:println(unpackedBytes); // @output [10,20,30,255]

    byte[] emptyBytes = [];
    pbany:Any packedEmptyBytes = check pbany:pack(emptyBytes);
    byte[] unpackedEmptyBytes = check pbany:unpack(packedEmptyBytes);
    io:println(unpackedEmptyBytes.length()); // @output 0
}
