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
    error plain = error("not a uuid error");
    io:println(plain is uuid:Error); // @output false

    string|error r = uuid:toString({
        timeLow: 1,
        timeMid: 2,
        timeHiAndVersion: 3,
        clockSeqHiAndReserved: 4,
        clockSeqLo: 5,
        node: 0x1000000000000
    });
    io:println(r is uuid:Error); // @output true

    r = uuid:toString({
        timeLow: 1,
        timeMid: 2,
        timeHiAndVersion: 0x1003,
        clockSeqHiAndReserved: 0x80,
        clockSeqLo: 4,
        node: -1
    });
    io:println(r is uuid:Error); // @output true

    io:println(check uuid:toString({
        timeLow: 1,
        timeMid: 2,
        timeHiAndVersion: 3,
        clockSeqHiAndReserved: 4,
        clockSeqLo: 5,
        node: 6
    })); // @output 00000001-0002-0003-0405-000000000006

    uuid:Uuid rec = check uuid:createType1AsRecord();
    io:println((rec.node >> 40) & 1); // @output 1
}
