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

import ballerina/time;

# Workflow execution context providing activity execution, durable sleep,
# deterministic time, and multi-future await APIs.
public client class Context {
    private handle nativeContext;

    # Creates a workflow execution context wrapping the native context handle.
    # This constructor is called by the workflow runtime; do not instantiate `Context` directly.
    # + nativeContext - Native context handle from the workflow engine
    public isolated function init(handle nativeContext) {
        self.nativeContext = nativeContext;
    }

    # Executes an activity function. A completed activity is never re-executed on replay — its
    # recorded result is reused — but a failed attempt can run again under a retry policy, so keep
    # side effects idempotent. `T` comes from the assignment, so bind every call: `() _ = check ...`
    # when the activity returns only `error?`.
    #
    # + activityFunction - The activity function (must have `@Activity`)
    # + args - Arguments keyed by parameter name. A module-level `final` client object may be
    #          passed where the activity declares one
    # + T - Expected return type (inferred from context)
    # + stepId - Identity of this step within the workflow, matching a node of the descriptor
    #            graph. A constant string; defaults to `<activity>#<ordinal>`
    # + options - How the invocation behaves: `approvalPolicy` gates the call behind a review;
    #             `retryPolicy` is `NoRetry` (default), `AutoRetry`, a `ReviewTaskDefinition` that
    #             raises a review on failure, or `RetryBeforeReview`
    # + return - The activity result as `T`, or an error
    remote isolated function callActivity(function activityFunction,
            map<anydata|object {}> args = {},
            typedesc<anydata> T = <>,
            string? stepId = (),
            *CallActivityOptions options)
            returns T|error = external;

    # Durable sleep that survives process crashes and restarts. Use instead of `runtime:sleep`.
    #
    # + duration - The duration to sleep
    # + stepId - Identity of this step within the workflow, as for `callActivity`
    # + return - An error if the sleep fails, otherwise nil
    public isolated function sleep(Duration duration, string? stepId = ()) returns error? {
        decimal totalSeconds = <decimal>duration.hours * 3600 +
                               <decimal>duration.minutes * 60 +
                               duration.seconds;
        int millis = <int>(totalSeconds * 1000);
        return sleepContextNative(self.nativeContext, millis, stepId);
    }

    # Returns the deterministic workflow time. Use instead of `time:utcNow()` inside workflows.
    #
    # + return - The current workflow time as `time:Utc`
    public isolated function currentTime() returns time:Utc {
        int millis = currentTimeMillisContextNative(self.nativeContext);
        int seconds = millis / 1000;
        decimal fraction = <decimal>(millis % 1000) / 1000d;
        return [seconds, fraction];
    }

    # Checks whether the workflow is recovering from a failure (re-executing recorded history).
    #
    # + return - `true` if recovering, `false` on first execution
    public isolated function isReplaying() returns boolean {
        return isReplayingNative(self.nativeContext);
    }

    # Get the unique workflow ID.
    #
    # + return - The workflow ID
    public isolated function getWorkflowId() returns string|error {
        return getWorkflowIdNative(self.nativeContext);
    }

    # Get the workflow type name.
    #
    # + return - The workflow type
    public isolated function getWorkflowType() returns string|error {
        return getWorkflowTypeNative(self.nativeContext);
    }

    # Who acted on the most recent human task this workflow created — or on the most recent one
    # with the given name. Lets a later task exclude or prefer that person. `()` before any task
    # has completed.
    #
    # + taskName - A task name, or `()` for the latest task of any name
    # + return - The completion, or `()`
    public isolated function lastHumanTaskCompletion(string? taskName = ()) returns HumanTaskCompletion? {
        return lastHumanTaskCompletionNative(self.nativeContext, taskName);
    }

    # The decision reached by the most recent review this workflow raised — a gated call or a
    # failed activity — or by the most recent review of the given task name. `()` before any
    # review has been decided.
    #
    # + taskName - A review task name, or `()` for the latest review of any name
    # + return - The decision, or `()`
    public isolated function lastReviewDecision(string? taskName = ()) returns ReviewDecisionRecord? {
        return lastReviewDecisionNative(self.nativeContext, taskName);
    }

    # Waits for at least `minCount` data futures to complete. Results are a positional tuple
    # aligned to input order; use nilable members (`T?`) for partial waits.
    #
    # + futures - Data futures from the workflow's events record
    # + minCount - Minimum completions required (default: all)
    # + timeout - Maximum wait duration; returns an error on timeout
    # + T - Expected return type, inferred from how the result is assigned
    # + return - Positional tuple of values (`nil` for an incomplete position), or an error
    remote isolated function await(future<anydata>[] futures,
            int:Unsigned32 minCount = <int:Unsigned32>futures.length(),
            Duration? timeout = (),
            typedesc<anydata|error|(anydata|error)[]> T = <>) returns T = external;

    # Creates a human task and blocks until a person completes it or the timeout elapses. The task
    # runs as a durable child workflow typed `taskName`, so it survives worker restarts.
    #
    # + taskName - Identifies the task type; used as the child workflow type and ID
    # + taskInput - Read-only object shown beside the form; `{}` when there is nothing to show
    # + T - Expected result type; drives form schema generation and runtime validation
    # + stepId - Identity of this step within the workflow, as for `callActivity`
    # + definition - The task's `HumanTaskDefinition`, as an included record (`userRoles = "MANAGER"`)
    # + return - The value the person submitted, or a `HumanTaskError` — timed out, rejected,
    #            or failed to produce a result
    remote isolated function awaitHumanTask(
            string taskName,
            map<json> taskInput,
            typedesc<anydata> T = <>,
            string? stepId = (),
            *HumanTaskDefinition definition)
            returns T|HumanTaskError = external;

    # Starts a child workflow and returns its instance ID without waiting. The child's lifecycle is
    # tied to this workflow, so in-flight children are cancelled when it closes. Read the result
    # later with `getChildWorkflowResult` or `waitForChildWorkflow` to fan out and gather.
    #
    # + childWorkflow - The child workflow function (must have `@Workflow`)
    # + input - Optional input for the child workflow. Must match the child workflow
    #           function's declared input parameter type (any `anydata` subtype)
    # + stepId - Identity of this step within the workflow, as for `callActivity`
    # + return - The child workflow instance ID, or an error if the child could not start
    remote isolated function runChildWorkflow(function childWorkflow, anydata input = (),
            string? stepId = ())
            returns string|error = external;

    # Returns a child workflow's result if it has already completed, without waiting. While the
    # child is still running this answers `WorkflowInProgressError` — check back later, or use the
    # blocking `waitForChildWorkflow`.
    #
    # + childWorkflowId - The child workflow instance ID returned by `runChildWorkflow`
    # + T - Expected result type (inferred from context)
    # + return - The child's result as `T`, a `workflow:WorkflowInProgressError` while the child
    #            is still running, or an error if the child failed
    remote isolated function getChildWorkflowResult(string childWorkflowId, typedesc<anydata> T = <>)
            returns T|error = external;

    # Waits durably until a child workflow started with `runChildWorkflow` completes and
    # returns its result. The wait is a durable suspend — no thread is held, and the wait
    # survives worker crashes and restarts (on replay the result is served from history).
    #
    # + childWorkflowId - The child workflow instance ID returned by `runChildWorkflow`
    # + T - Expected result type (inferred from context)
    # + return - The child's result as `T`, or an error if the child failed
    remote isolated function waitForChildWorkflow(string childWorkflowId, typedesc<anydata> T = <>)
            returns T|error = external;

    # Starts a child workflow and durably waits for its result — `runChildWorkflow` followed by
    # `waitForChildWorkflow` in one call. The wait is a durable suspend, not a held thread.
    #
    # + childWorkflow - The child workflow function (must have `@Workflow`)
    # + input - Optional input for the child workflow. Must match the child workflow
    #           function's declared input parameter type (any `anydata` subtype)
    # + T - Expected result type (inferred from context)
    # + return - The child's result as `T`, or an error if the child failed
    remote isolated function callWorkflow(function childWorkflow, anydata input = (),
            typedesc<anydata> T = <>, string? stepId = ()) returns T|error = external;

    # Sends data to a running workflow instance's events record from inside a workflow — the
    # in-workflow counterpart of `workflow:sendData`, usually aimed at a child workflow.
    #
    # + childWorkflowId - Target workflow instance ID (usually from `runChildWorkflow`)
    # + dataName - Field name in the target workflow's events record
    # + data - The data payload
    # + return - An error if sending fails
    remote isolated function sendDataToChildWorkflow(string childWorkflowId, string dataName,
            anydata data) returns error? = external;
}

// Native function declarations

isolated function sleepContextNative(handle contextHandle, int millis, string? stepId) returns error? = external;

isolated function currentTimeMillisContextNative(handle contextHandle) returns int = external;

isolated function isReplayingNative(handle contextHandle) returns boolean = external;

isolated function getWorkflowIdNative(handle contextHandle) returns string|error = external;

isolated function lastHumanTaskCompletionNative(handle contextHandle, string? taskName)
        returns HumanTaskCompletion? = external;

isolated function lastReviewDecisionNative(handle contextHandle, string? taskName)
        returns ReviewDecisionRecord? = external;

isolated function getWorkflowTypeNative(handle contextHandle) returns string|error = external;

isolated function newContext(handle nativeContext) returns Context => new (nativeContext);
