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
import ballerina/uuid;

public function main() returns error? {
    // toRecord / toString round-trip with a known UUID
    string known = "550e8400-e29b-41d4-a716-446655440000";

    uuid:Uuid rec = check uuid:toRecord(known);

    string back = check uuid:toString(rec);
    io:println(back); // @output 550e8400-e29b-41d4-a716-446655440000
    
    // toBytes round-trip
    byte[]|uuid:Error bytes = uuid:toBytes(known);
    io:println(bytes is byte[]); // @output true
    if bytes is byte[] {
        io:println(bytes.length()); // @output 16
        string|error fromBytes = uuid:toString(bytes);
        if fromBytes is string {
            io:println(fromBytes); // @output 550e8400-e29b-41d4-a716-446655440000
        }
    }

    // getVersion returns V4 for the known UUID (version nibble = 4)
    uuid:Version|uuid:Error ver = check uuid:getVersion(known);
    io:println(ver == uuid:V4); // @output true

    // toString / toRecord reject byte arrays that aren't exactly 16 bytes
    byte[] shortArr = [5, 12, 16, 35];
    string|error shortToString = uuid:toString(shortArr);
    io:println(shortToString is error); // @output true

    uuid:Uuid|uuid:Error shortToRecord = uuid:toRecord(shortArr);
    io:println(shortToRecord is uuid:Error); // @output true

    byte[] longArr = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20];
    string|error longToString = uuid:toString(longArr);
    io:println(longToString is error); // @output true

    uuid:Uuid|uuid:Error longToRecord = uuid:toRecord(longArr);
    io:println(longToRecord is uuid:Error); // @output true
}
