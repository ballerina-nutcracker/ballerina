// Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

# Starts a new workflow instance and returns its unique ID.
#
# + processFunction - The workflow function (must have `@Workflow`)
# + input - Optional input data for the workflow. Must match the workflow
#           function's declared input parameter type (any `anydata` subtype)
# + return - The workflow ID, or an error
public isolated function run(function processFunction, anydata input = ()) returns string|error {
    return runNative(processFunction, input);
}

isolated function runNative(function processFunction, anydata input) returns string|error = external;

# Starts a workflow instance under an id the caller chooses — a business key such as an order
# number — so the id doubles as the correlation key. The policies say what happens when the id is
# already held: `ifRunning = USE_EXISTING` makes a retried submit join the instance it already
# started instead of failing. The id must not be blank, longer than 255 bytes in UTF-8, or start with
# a reserved prefix (`humantask-`, `reviewactivity-`, `childwf-`, `childagent-`).
#
# + processFunction - The workflow function (must have `@Workflow`)
# + instanceId - The instance id to start under
# + input - Optional input data for the workflow, as for `run`
# + ifRunning - What to do when an instance with this id is running
# + ifClosed - What to do when an instance with this id has closed
# + return - The instance id (the same string), an `InstanceAlreadyExistsError` when the id is
#            held and the policy refused, or an error
public isolated function runWithId(function processFunction, string instanceId, anydata input = (),
        RunningInstancePolicy ifRunning = FAIL, ClosedInstancePolicy ifClosed = ALLOW_DUPLICATE)
        returns string|InstanceAlreadyExistsError|error {
    return runWithIdNative(processFunction, instanceId, input, ifRunning, ifClosed);
}

isolated function runWithIdNative(function processFunction, string instanceId, anydata input,
        string ifRunning, string ifClosed) returns string|error = external;

# Sends data to a running workflow's events record.
#
# + workflow - The workflow function (must have `@Workflow`)
# + workflowId - Target workflow ID (from `run`)
# + dataName - Field name in the workflow's events record
# + data - The data payload
# + return - An error if sending fails
public isolated function sendData(function workflow, string workflowId, string dataName, anydata data) returns error? {
    return sendDataNative(workflow, workflowId, dataName, data);
}

isolated function sendDataNative(function workflow, string workflowId, string dataName,
        anydata data) returns error? = external;

# Lists the data events a running durable agent has accepted but not yet answered. Use it after a
# restart to rediscover in-flight turns and read their answers with `DurableAgent.getDataResult`
# or `waitForDataResult`.
#
# + agentId - Target agent (workflow) ID
# + return - The in-flight event turns (empty when the agent is idle), or an error
public isolated function getPendingAgentEvents(string agentId)
        returns PendingAgentEvent[]|error = external;

# Returns an instance's result if it has finished, without waiting. While the instance runs —
# executing, or suspended on a human task — a `WorkflowInProgressError` is returned: check back
# later, or use `waitForResult`. An instance that closed without a result answers
# `InstanceFailedError`; an id nothing holds, `InstanceNotFoundError`.
#
# + instanceId - The instance id `run` returned
# + T - Expected result type (inferred from context)
# + return - The result as `T`, or one of the read errors
public isolated function getResult(string instanceId, typedesc<anydata> T = <>)
        returns T|error = external;

# Waits for an instance to finish and returns its result. With a `timeout` the wait is bounded,
# and a `WorkflowInProgressError` says the instance outlived it — the same answer `getResult`
# gives — so a caller can wait a little and then report progress. Without one the wait lasts as
# long as the instance does. Crash-resumable: the result lives in history, so the call can be
# made again after a restart.
#
# + instanceId - The instance id `run` returned
# + T - Expected result type (inferred from context)
# + timeout - The longest to wait, or `()` for as long as it takes; give it by name
# + return - The result as `T`, or one of the read errors
public isolated function waitForResult(string instanceId, typedesc<anydata> T = <>, Duration? timeout = ())
        returns T|error = external;

# Returns an instance's status without waiting or reading its history.
#
# + instanceId - The instance id `run` returned
# + return - The status, or an `InstanceNotFoundError`
public isolated function getStatus(string instanceId) returns InstanceStatus|InstanceNotFoundError|error
        = external;

# Completes a pending human task by sending the result back to the waiting workflow. Given
# `callerRoles`, the caller must hold one of the task's `userRoles`; omitting them skips the check
# and leaves enforcement to the caller.
#
# + taskWorkflowId - Temporal workflow ID of the human task child workflow
# + result - The value to return to the workflow (must be compatible with the declared `T`)
# + callerRoles - Roles held by the caller; validated against the task's configured `userRoles`
# + userId - The user ID of the person completing the task (used for auditing)
# + return - An error if the task cannot be found, is already completed, or the caller is unauthorized
public isolated function completeHumanTask(string taskWorkflowId, anydata result,
        [string, string...]? callerRoles = (), string? userId = ()) returns error? {
    map<anydata>|error receipt = completeHumanTaskNative(taskWorkflowId, result, callerRoles, userId);
    if receipt is error {
        return receipt;
    }
}

// On success the runtime hands back what it confirmed about the task — its declared name, its
// parent workflow and the roles allowed to decide it — for the decision's audit entry.
isolated function completeHumanTaskNative(string taskWorkflowId, anydata result,
        [string, string...]? callerRoles, string? userId) returns map<anydata>|error = external;

// Internal functions

# Stops the workflow runtime gracefully, draining in-progress tasks.
#
# + return - An error if stopping fails
isolated function stopWorkflowRuntime() returns error? = external;

// Factories for the engine: Go code cannot construct the module's typed errors itself.
isolated function newInProgressError(string instanceId) returns WorkflowInProgressError =>
    error WorkflowInProgressError("workflow instance is still running: " + instanceId);

isolated function newInstanceNotFoundError(string instanceId) returns InstanceNotFoundError =>
    error InstanceNotFoundError("workflow instance not found: " + instanceId, instanceId = instanceId);

isolated function newInstanceFailedError(string instanceId, error cause) returns InstanceFailedError =>
    error InstanceFailedError(cause.message(), cause, instanceId = instanceId, status = FAILED);

isolated function newHumanTaskFailedError(string message) returns HumanTaskFailedError =>
    error HumanTaskFailedError(message);
