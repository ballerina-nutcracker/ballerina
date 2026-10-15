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
import ballerina/lang.runtime;
import ballerina/task;
import ballerina/time;

isolated map<int> counts = {};

isolated function count(string name) returns int {
    lock {
        return counts[name] ?: 0;
    }
}

isolated service class Counter {
    *task:Service;
    private final string name;

    isolated function init(string name) {
        self.name = name;
    }

    public isolated function execute() returns error? {
        lock {
            counts[self.name] = (counts[self.name] ?: 0) + 1;
        }
    }
}

class OkJob {
    *task:Job;

    public function execute() {
    }
}

public function main() returns error? {
    task:Listener l = check new (trigger = {interval: 0.05, maxCount: 2});
    Counter a = new ("a");
    Counter b = new ("b");
    check l.attach(a, "a");
    check l.attach(b, ["b", "ignored"]);
    check l.'start();
    task:Error? e = l.'start();
    io:println(e is task:Error ? e.message() : "started"); // @output Unable to store Job : 'DEFAULT.a', because one already exists with this identification.

    task:Listener other = check new (trigger = {interval: 0.05, maxCount: 2});
    check other.attach(new Counter("x"), "a");
    e = other.'start();
    io:println(e is task:Error ? e.message() : "started"); // @output Unable to store Job : 'DEFAULT.a', because one already exists with this identification.

    runtime:sleep(0.3);
    io:println("a: ", count("a"), " b: ", count("b"), " x: ", count("x")); // @output a: 2 b: 2 x: 0
    Counter late = new ("late");
    check l.attach(late, "late");
    runtime:sleep(0.2);
    io:println("attached after start: ", count("late")); // @output attached after start: 0
    io:println(l.detach(a) is ()); // @output true
    e = l.detach(a);
    io:println(e is task:Error ? e.message() : "detached"); // @output Service is not found in the listener
    io:println(l.gracefulStop() is ()); // @output true

    task:Listener unnamed = check new (trigger = {interval: 0.05, maxCount: 1});
    check unnamed.attach(new Counter("unnamed"));
    check unnamed.'start();
    runtime:sleep(0.2);
    io:println("unnamed: ", count("unnamed")); // @output unnamed: 1
    io:println(unnamed.immediateStop() is ()); // @output true

    task:Listener forever = check new (trigger = {interval: 0.05, maxCount: 0});
    Counter s1 = new ("s1");
    Counter s2 = new ("s2");
    check forever.attach(s1, "s1");
    check forever.attach(s2, "s2");
    check forever.'start();
    task:JobId job = check task:scheduleOneTimeJob(new OkJob(), time:utcToCivil(time:utcAddSeconds(time:utcNow(), 5)));
    runtime:sleep(0.2);
    io:println("maxCount 0 repeats: ", count("s1") > 2); // @output maxCount 0 repeats: true
    check task:unscheduleJob(job);
    int stoppedAt = count("s1");
    runtime:sleep(0.2);
    io:println("services stop with the scheduler: ", count("s1") == stoppedAt); // @output services stop with the scheduler: true
    e = forever.detach(s1);
    io:println(e is task:Error ? e.message() : "detached"); // @output The Scheduler has been shutdown.
    e = forever.gracefulStop();
    io:println(e is task:Error ? e.message() : "stopped"); // @output The Scheduler has been shutdown.

    task:Listener zero = check new (trigger = {interval: 0});
    check zero.attach(new Counter("zero"), "zero");
    e = zero.'start();
    io:println(e is task:Error ? e.message() : "started"); // @output Repeat Interval cannot be zero.
    decimal negativeInterval = -1;
    task:Listener negative = check new (trigger = {interval: negativeInterval});
    check negative.attach(new Counter("negative"), "negative");
    e = negative.'start();
    io:println(e is task:Error ? e.message() : "started"); // @output Repeat interval must be >= 0
    task:Listener coordinated = check new (trigger = {interval: 1},
        warmBackupConfig = {taskId: "node-1", groupId: "group-1"});
    check coordinated.attach(new Counter("coordinated"), "coordinated");
    e = coordinated.'start();
    io:println(e is task:Error ? e.message() : "started"); // @output Task coordination through warmBackupConfig is not supported
}
