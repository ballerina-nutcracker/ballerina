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
import ballerina/workflow;

@workflow:Workflow
isolated function slowApprove(workflow:Context ctx, string input, record {|future<boolean> approval;|} events)
        returns string|error {
    check ctx.sleep({seconds: 0.05});
    boolean ok = check wait events.approval;
    return input + (ok ? " approved" : " rejected");
}

@workflow:Workflow
isolated function badSleep(workflow:Context ctx, string input) returns string|error {
    check ctx.sleep({months: 1});
    return input;
}

// Not isolated, so it shares main's thread: a wait that held the thread would never let it run
function approve(string id) returns error? {
    check workflow:sendData(slowApprove, id, "approval", true);
}

public function main() returns error? {
    string id = check workflow:run(slowApprove, "o1");
    string|error early = workflow:waitForResult(id, timeout = {seconds: 0.01});
    io:println(early is workflow:WorkflowInProgressError); // @output true
    string|error negative = workflow:waitForResult(id, timeout = {seconds: -1});
    io:println(negative is error ? negative.message() : negative); // @output a timeout must not be negative
    string|error months = workflow:waitForResult(id, timeout = {months: 1});
    io:println(months is error ? months.message() : months); // @output a timeout does not support 'years' or 'months': they have no fixed length
    future<error?> f = start approve(id);
    io:println(check workflow:waitForResult(id)); // @output o1 approved
    check wait f;

    string|error slept = workflow:waitForResult(check workflow:run(badSleep, "x"));
    io:println(slept is workflow:InstanceFailedError); // @output true
    io:println(slept is error ? slept.message() : slept); // @output sleep does not support 'years' or 'months': they have no fixed length
}
