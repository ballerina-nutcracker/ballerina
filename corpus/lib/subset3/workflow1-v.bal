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

type Order record {|
    string id;
    int amount;
|};

type Approval record {|
    boolean ok;
    string who;
|};

@workflow:Activity
isolated function charge(Order item, int fee) returns int|error {
    return item.amount + fee;
}

@workflow:Workflow
isolated function processOrder(workflow:Context ctx, Order input, record {|future<Approval> approval;|} events)
        returns string|error {
    int total = check ctx->callActivity(charge, {item: input, fee: 5});
    Approval approval = check wait events.approval;
    if approval.ok {
        return input.id + " charged " + total.toString() + " by " + approval.who;
    }
    return input.id + " declined by " + approval.who;
}

public function main() returns error? {
    string a = check workflow:run(processOrder, {id: "o-1", amount: 10});
    string b = check workflow:run(processOrder, {id: "o-2", amount: 20});
    string|error early = workflow:getResult(a);
    io:println(early is workflow:WorkflowInProgressError); // @output true
    io:println(check workflow:getStatus(a)); // @output RUNNING
    error? refused = workflow:sendData(processOrder, a, "approval", {ok: "yes"});
    io:println(refused is error); // @output true
    check workflow:sendData(processOrder, b, "approval", {ok: false, who: "bob"});
    check workflow:sendData(processOrder, a, "approval", {ok: true, who: "ann"});
    io:println(check workflow:waitForResult(a)); // @output o-1 charged 15 by ann
    io:println(check workflow:waitForResult(b)); // @output o-2 declined by bob
    io:println(check workflow:getStatus(a)); // @output COMPLETED
    io:println(check workflow:getResult(b)); // @output o-2 declined by bob
}
