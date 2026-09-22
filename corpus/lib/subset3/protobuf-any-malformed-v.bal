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
    // A hand-built Any with a non-hex value fails to decode.
    pbany:Any badHex = {typeUrl: "type.googleapis.com/google.protobuf.BoolValue", value: "not-hex!"};
    boolean|error unpackResult = pbany:unpack(badHex);
    io:println(unpackResult is error); // @output true
    if unpackResult is error {
        io:println(unpackResult is pbany:Error); // @output true
    }

    // A map<anydata> containing a leaf value with no google.protobuf.Struct representation
    // (xml is anydata, but not a representable Struct field) fails to pack.
    xml x = xml `<a/>`;
    map<anydata> withXml = {"x": x};
    pbany:Any|error packResult = pbany:pack(withXml);
    io:println(packResult is error); // @output true
    if packResult is error {
        io:println(packResult is pbany:Error); // @output true
    }
}
