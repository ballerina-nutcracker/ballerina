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

import ballerina/cache;
import ballerina/io;
import ballerina/lang.runtime;
import ballerina/time;

function sleepToFraction(decimal fraction) {
    time:Utc now = time:utcNow();
    runtime:sleep(1d + fraction - now[1]);
}

public function main() returns error? {
    cache:Cache c = new (capacity = 10, defaultMaxAge = 1.5);
    // Expiry compares whole seconds, so an entry expires within the second it is due in.
    sleepToFraction(0.1);
    check c.put("half", "v", 0.5);
    check c.put("forever", "v");
    check c.put("zero", "v", 0);
    check c.put("negative", "v", -5);
    check c.put("long", "v", 100);
    runtime:sleep(0.2);
    io:println(c.get("half") is cache:Error); // @output true
    io:println(c.get("zero") is cache:Error, " ", c.get("negative") is cache:Error); // @output false false
    runtime:sleep(1.5);
    io:println(c.get("zero") is cache:Error, " ", c.get("negative") is cache:Error); // @output true true
    io:println(c.get("long") is cache:Error); // @output false

    cache:Cache forever = new (capacity = 10);
    check forever.put("a", "v");
    check forever.put("b", "v", -1);
    check forever.put("c", "v", 1);
    runtime:sleep(2);
    io:println(forever.size(), " ", forever.hasKey("c")); // @output 3 true
    io:println(forever.get("a") is cache:Error, " ", forever.get("c") is cache:Error); // @output false true
    io:println(forever.size()); // @output 2

    cache:Cache cleaned = new (capacity = 10, defaultMaxAge = 1, cleanupInterval = 0.5);
    check cleaned.put("a", 1);
    check cleaned.put("b", 2);
    check cleaned.put("kept", 3, 100);
    runtime:sleep(2.5);
    io:println(cleaned.size(), " ", cleaned.keys()); // @output 1 ["kept"]
}
