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
import ballerina/time;
import ballerina/protobuf.types.'any as pbany;

public function main() returns error? {
    // A time:Seconds value whose whole-seconds part does not fit in an int64 cannot be
    // packed: google.protobuf.Duration stores seconds as int64.
    time:Seconds tooLarge = 1e6144;
    pbany:Any|pbany:Error packedTooLarge = pbany:pack(tooLarge);
    io:println(packedTooLarge is pbany:Error); // @output true

    // Type mismatch: a packed boolean cannot unpack as a string.
    pbany:Any packedBool = check pbany:pack(true);
    string|error mismatch = pbany:unpack(packedBool);
    io:println(mismatch is error); // @output true
    io:println(mismatch is pbany:TypeMismatchError); // @output true
    io:println(mismatch is pbany:Error); // @output true
    if mismatch is error {
        io:println(mismatch.message()); // @output Type type.googleapis.com/google.protobuf.BoolValue cannot unpack to string
    }

    // An Any value with a typeUrl this module does not recognize is also a TypeMismatchError,
    // not a generic decode failure.
    pbany:Any bogus = {typeUrl: "type.googleapis.com/google.protobuf.Bogus", value: ""};
    int|error bogusResult = pbany:unpack(bogus);
    io:println(bogusResult is pbany:TypeMismatchError); // @output true

    // Distinct-error identity.
    pbany:Error baseErr = error pbany:Error("boom");
    io:println(baseErr is pbany:Error); // @output true
    io:println(baseErr is pbany:TypeMismatchError); // @output false

    pbany:TypeMismatchError specificErr = error pbany:TypeMismatchError("mismatch");
    io:println(specificErr is pbany:Error); // @output true
    io:println(specificErr is pbany:TypeMismatchError); // @output true
}
