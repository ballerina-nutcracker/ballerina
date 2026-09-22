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
import ballerina/protobuf.types.timestamp;
import ballerina/protobuf.types.'any as pbany;

public function main() returns error? {
    time:Utc ts = check time:utcFromString("2024-01-01T00:00:00.500Z");
    timestamp:ContextTimestamp ctxTs = {content: ts, headers: {}};
    io:println(ctxTs.content[0]); // @output 1704067200

    pbany:Any packedTs = check pbany:pack(ts);
    io:println(packedTs.typeUrl); // @output type.googleapis.com/google.protobuf.Timestamp
    time:Utc unpackedTs = check pbany:unpack(packedTs);
    io:println(unpackedTs[0]); // @output 1704067200
    io:println(unpackedTs[1]); // @output 0.5

    time:Utc epoch = [0, 0d];
    pbany:Any packedEpoch = check pbany:pack(epoch);
    time:Utc unpackedEpoch = check pbany:unpack(packedEpoch);
    io:println(unpackedEpoch[0]); // @output 0
}
