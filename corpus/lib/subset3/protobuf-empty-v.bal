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
import ballerina/protobuf.types.empty;
import ballerina/protobuf.types.'any as pbany;

public function main() returns error? {
    empty:Empty e = {};
    io:println(e is empty:Empty); // @output true

    empty:ContextNil nilCtx = {headers: {"x-id": "1"}};
    io:println(nilCtx.headers["x-id"]); // @output 1

    pbany:Any packedNil = check pbany:pack(());
    io:println(packedNil.typeUrl); // @output type.googleapis.com/google.protobuf.Empty
    () unpackedNil = check pbany:unpack(packedNil);
    io:println(unpackedNil); // @output

    // A closed empty record needs no proto descriptor, so it keeps the Empty mapping
    // instead of falling through to the Struct fallback used for other records.
    pbany:Any packedEmptyRec = check pbany:pack(e);
    io:println(packedEmptyRec.typeUrl); // @output type.googleapis.com/google.protobuf.Empty
    io:println(packedEmptyRec.value == ""); // @output true

    // A genuine empty map is still a map, not a closed empty record, so it stays Struct.
    map<anydata> emptyMap = {};
    pbany:Any packedEmptyMap = check pbany:pack(emptyMap);
    io:println(packedEmptyMap.typeUrl); // @output type.googleapis.com/google.protobuf.Struct
}
