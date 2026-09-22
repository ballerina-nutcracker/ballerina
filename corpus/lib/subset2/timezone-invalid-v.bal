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

public function main() returns error? {
    // Invalid zone ID returns an error from init
    time:TimeZone|error tz = new time:TimeZone("Invalid/Zone");
    io:println(tz is error); // @output true

    // utcFromCivil without timeAbbrev returns an error
    time:TimeZone utcZone = check new time:TimeZone("UTC");
    time:Civil civil = {year: 2021, month: 4, day: 12, hour: 10, minute: 0};
    time:Utc|error result = utcZone.utcFromCivil(civil);
    io:println(result is error); // @output true
}
