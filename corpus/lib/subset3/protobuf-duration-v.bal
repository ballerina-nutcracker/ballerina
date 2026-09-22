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
import ballerina/protobuf.types.duration;
import ballerina/protobuf.types.'any as pbany;

public function main() returns error? {
    duration:ContextDuration ctxDur = {content: 12.5, headers: {}};
    io:println(ctxDur.content); // @output 12.5

    time:Seconds dur = 12.5;
    pbany:Any packedDur = check pbany:pack(dur);
    io:println(packedDur.typeUrl); // @output type.googleapis.com/google.protobuf.Duration
    time:Seconds unpackedDur = check pbany:unpack(packedDur);
    io:println(unpackedDur); // @output 12.5

    // Nanosecond-precision round-trip.
    time:Seconds precise = 12.123456789;
    pbany:Any packedPrecise = check pbany:pack(precise);
    time:Seconds unpackedPrecise = check pbany:unpack(packedPrecise);
    io:println(unpackedPrecise); // @output 12.123456789

    time:Seconds zero = 0;
    pbany:Any packedZero = check pbany:pack(zero);
    time:Seconds unpackedZero = check pbany:unpack(packedZero);
    io:println(unpackedZero); // @output 0

    // A timestamp-scale value with a fraction just under a whole second needs 19
    // significant digits, more than float64 carries. The wire bytes are asserted
    // directly because a round-trip alone still succeeds when the split is wrong:
    // seconds must stay 1735689599 with nanos 999999999, not 1735689600 with -1,
    // which google.protobuf.Duration forbids for being oppositely signed.
    time:Seconds bigSeconds = 1735689599.999999999;
    pbany:Any packedBig = check pbany:pack(bigSeconds);
    io:println(packedBig.value); // @output 08FF8AD2BB0610FF93EBDC03
    time:Seconds unpackedBig = check pbany:unpack(packedBig);
    io:println(unpackedBig); // @output 1735689599.999999999

    // The same split, truncating toward zero, keeps both parts negative.
    time:Seconds negBigSeconds = -1735689599.999999999;
    pbany:Any packedNeg = check pbany:pack(negBigSeconds);
    io:println(packedNeg.value); // @output 0881F5ADC4F9FFFFFFFF011081EC94A3FCFFFFFFFF01
    time:Seconds unpackedNeg = check pbany:unpack(packedNeg);
    io:println(unpackedNeg); // @output -1735689599.999999999
}
