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

isolated int calls = 0;

isolated function hit() returns int {
    lock {
        calls += 1;
        return calls;
    }
}

isolated function callCount() returns int {
    lock {
        return calls;
    }
}

isolated service class Panicky {
    *task:Service;

    public isolated function execute() returns error? {
        panic error("service panic");
    }
}

isolated service class RetryPanicky {
    *task:Service;

    public isolated function execute() returns error? {
        if hit() == 1 {
            return error("fail");
        }
        panic error("retry panic");
    }
}

public function main() returns error? {
    task:Listener plain = check new (trigger = {interval: 1, maxCount: 1});
    check plain.attach(new Panicky(), "plain");
    check plain.'start();
    runtime:sleep(0.3);
    check plain.gracefulStop();

    task:Listener retrying = check new (trigger = {interval: 1, maxCount: 1,
        taskPolicy: {errorPolicy: task:CONTINUE}, retryConfig: {maxAttempts: 3, retryInterval: 0}});
    check retrying.attach(new RetryPanicky(), "retrying");
    check retrying.'start();
    runtime:sleep(0.3);
    io:println("calls: ", callCount()); // @output calls: 2
    check retrying.gracefulStop();
}
