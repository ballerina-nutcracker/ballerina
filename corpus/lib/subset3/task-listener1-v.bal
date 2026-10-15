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
import ballerina/task;
import ballerina/time;

isolated function inSeconds(decimal seconds) returns time:Civil {
    return time:utcToCivil(time:utcAddSeconds(time:utcNow(), seconds));
}

public function main() returns error? {
    task:Listener idle = check new (trigger = {interval: 1});
    io:println(idle.'start() is ()); // @output true
    io:println(idle.'start() is ()); // @output true
    io:println(idle.gracefulStop() is ()); // @output true
    io:println(idle.immediateStop() is ()); // @output true

    task:Listener windowed = check new (trigger = {interval: 1, maxCount: 3, startTime: inSeconds(5), endTime: inSeconds(10)});
    io:println(windowed.'start() is ()); // @output true
    io:println(windowed.gracefulStop() is ()); // @output true

    time:Civil past = {year: 2020, month: 1, day: 1, hour: 0, minute: 0, second: 0};
    task:Listener pastStart = check new (trigger = {interval: 1, startTime: past});
    task:Error? e = pastStart.'start();
    io:println(e is task:Error ? e.message().substring(0, 53) : "started"); // @output Error converting civil to milliseconds: Invalid time:
    task:Listener pastEnd = check new (trigger = {interval: 1, endTime: past});
    e = pastEnd.'start();
    io:println(e is task:Error ? e.message().substring(0, 53) : "started"); // @output Error converting civil to milliseconds: Invalid time:

    task:Listener coordinated = check new (trigger = {interval: 1},
        warmBackupConfig = {taskId: "node-1", groupId: "group-1"});
    e = coordinated.'start();
    io:println(e is task:Error ? e.message() : "started"); // @output started

    check task:configureWorkerPool(0, 5);
    e = idle.'start();
    io:println(e is task:Error ? e.message() : "started"); // @output Cannot create the Scheduler.Thread count must be > 0
    check task:configureWorkerPool();
    io:println(idle.'start() is ()); // @output true
}
