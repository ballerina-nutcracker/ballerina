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

@workflow:Activity
isolated function divide(int d) returns int {
    return 10 / d;
}

// An activity's panic reaches the workflow as an error
@workflow:Workflow
isolated function safeDivide(workflow:Context ctx, int input) returns int|error {
    int|error r = ctx->callActivity(divide, {d: input});
    return r is error ? -1 : r;
}

@workflow:Workflow
isolated function explode(workflow:Context ctx, int input) returns int|error {
    _ = check ctx.getWorkflowId();
    if input == 1 {
        panic error("workflow panicked");
    }
    return error("declined " + input.toString());
}

@workflow:Workflow
isolated function misnamed(workflow:Context ctx, int input) returns int|error {
    return ctx->callActivity(divide, {divisor: input});
}

public function main() returns error? {
    io:println(check workflow:waitForResult(check workflow:run(safeDivide, 0))); // @output -1
    io:println(check workflow:waitForResult(check workflow:run(safeDivide, 2))); // @output 5

    int|error panicked = workflow:waitForResult(check workflow:run(explode, 1));
    io:println(panicked is workflow:InstanceFailedError); // @output true
    io:println(panicked is error ? panicked.message() : "?"); // @output workflow panicked

    string declinedId = check workflow:run(explode, 2);
    int|error declined = workflow:waitForResult(declinedId);
    io:println(declined is workflow:InstanceFailedError); // @output true
    io:println(declined is error ? declined.message() : "?"); // @output declined 2
    io:println(check workflow:getStatus(declinedId)); // @output FAILED

    int|error bad = workflow:waitForResult(check workflow:run(misnamed, 2));
    io:println(bad is error ? bad.message() : "?"); // @output unknown activity argument: divisor

    string|error none = workflow:getResult("no-such-id");
    io:println(none is workflow:InstanceNotFoundError); // @output true
    io:println(workflow:getStatus("no-such-id") is workflow:InstanceNotFoundError); // @output true
    io:println(workflow:sendData(explode, "no-such-id", "x", 1) is workflow:InstanceNotFoundError); // @output true

    string|error withId = workflow:runWithId(explode, "chosen", 2);
    io:println(withId is error); // @output true
}
