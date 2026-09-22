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
import ballerina/time;

public function main() {
    // getZone returns a Zone for a valid ID
    time:Zone? zone = time:getZone("Asia/Colombo");
    io:println(zone != ()); // @output true

    // getZone returns nil for an invalid ID
    time:Zone? bad = time:getZone("Not/AZone");
    io:println(bad == ()); // @output true

    // getZone with a fixed-offset ID
    time:Zone? fixedZone = time:getZone("+05:30");
    io:println(fixedZone != ()); // @output true
}
