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

isolated map<int> counts = {};

isolated function hit(string name) returns int {
    lock {
        int n = (counts[name] ?: 0) + 1;
        counts[name] = n;
        return n;
    }
}

isolated function count(string name) returns int {
    lock {
        return counts[name] ?: 0;
    }
}

listener task:Listener limited = new (trigger = {interval: 0.05, maxCount: 3});

service "limited" on limited {
    public isolated function execute() {
        _ = hit("limited");
    }
}

listener task:Listener terminating = new (trigger = {interval: 0.05, maxCount: 3, taskPolicy: {errorPolicy: task:TERMINATE}});

service "terminate" on terminating {
    public isolated function execute() returns error? {
        _ = hit("terminate");
        return error("service failed");
    }
}

listener task:Listener continuing = new (trigger = {interval: 0.05, maxCount: 3, taskPolicy: {errorPolicy: task:CONTINUE}});

service on continuing {
    public isolated function execute() returns error? {
        _ = hit("continue");
        return error("service failed");
    }
}

listener task:Listener fixedRetry = new (trigger = {
    interval: 1,
    maxCount: 1,
    taskPolicy: {errorPolicy: task:CONTINUE},
    retryConfig: {maxAttempts: 2, retryInterval: 0}
});

service "fixedRetry" on fixedRetry {
    public isolated function execute() returns error? {
        _ = hit("fixedRetry");
        return error("still failing");
    }
}

listener task:Listener exponentialRetry = new (trigger = {
    interval: 1,
    maxCount: 1,
    taskPolicy: {errorPolicy: task:TERMINATE},
    retryConfig: {maxAttempts: 3, retryInterval: 0, backoffStrategy: task:EXPONENTIAL, maxInterval: 0}
});

service "exponentialRetry" on exponentialRetry {
    public isolated function execute() returns error? {
        _ = hit("exponentialRetry");
        return error("still failing");
    }
}

listener task:Listener unboundedRetry = new (trigger = {
    interval: 1,
    maxCount: 1,
    taskPolicy: {errorPolicy: task:CONTINUE},
    retryConfig: {maxAttempts: 2, retryInterval: 0, backoffStrategy: task:EXPONENTIAL}
});

service "unboundedRetry" on unboundedRetry {
    public isolated function execute() returns error? {
        _ = hit("unboundedRetry");
        return error("still failing");
    }
}

listener task:Listener recovered = new (trigger = {
    interval: 1,
    maxCount: 1,
    retryConfig: {maxAttempts: 3, retryInterval: 0}
});

service "recovered" on recovered {
    public isolated function execute() returns error? {
        if hit("recovered") < 2 {
            return error("first attempt fails");
        }
    }
}

listener task:Listener cappedRetry = new (trigger = {
    interval: 1,
    maxCount: 1,
    taskPolicy: {errorPolicy: task:CONTINUE},
    retryConfig: {maxAttempts: 3, retryInterval: 0, maxInterval: -1}
});

service "cappedRetry" on cappedRetry {
    public isolated function execute() returns error? {
        _ = hit("cappedRetry");
        return error("still failing");
    }
}

listener task:Listener slowRetry = new (trigger = {
    interval: 1,
    maxCount: 1,
    taskPolicy: {errorPolicy: task:CONTINUE},
    retryConfig: {maxAttempts: 3, retryInterval: 5}
});

service "slowRetry" on slowRetry {
    public isolated function execute() returns error? {
        _ = hit("slowRetry");
        return error("still failing");
    }
}

// testMain runs after the listeners have started.
public function testMain() {
    runtime:sleep(0.5);
    io:println("limited: ", count("limited")); // @output limited: 3
    io:println("terminate: ", count("terminate")); // @output terminate: 1
    io:println("continue: ", count("continue")); // @output continue: 3
    io:println("fixedRetry: ", count("fixedRetry")); // @output fixedRetry: 3
    io:println("exponentialRetry: ", count("exponentialRetry")); // @output exponentialRetry: 4
    io:println("unboundedRetry: ", count("unboundedRetry")); // @output unboundedRetry: 3
    io:println("recovered: ", count("recovered")); // @output recovered: 2
    io:println("cappedRetry: ", count("cappedRetry")); // @output cappedRetry: 1
    io:println("slowRetry: ", count("slowRetry")); // @output slowRetry: 1
}
