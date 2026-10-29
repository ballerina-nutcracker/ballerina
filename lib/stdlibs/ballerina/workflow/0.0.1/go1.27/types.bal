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

# Deployment mode for the workflow runtime.
#
# + LOCAL - Local development server (e.g., `temporal server start-dev`)
# + CLOUD - Managed cloud deployment (requires authentication)
# + SELF_HOSTED - Self-hosted server (authentication is optional)
# + IN_MEMORY - Lightweight in-memory engine (no persistence, no external server)
public enum Mode {
    LOCAL,
    CLOUD,
    SELF_HOSTED,
    IN_MEMORY
}

# Internal retry policy used to pass module-level defaults to the native layer.
# + initialIntervalInSeconds - Initial delay before the first retry attempt in seconds
# + backoffCoefficient - Multiplier applied to the interval after each retry
# + maximumIntervalInSeconds - Optional cap on the delay between retries in seconds
# + maximumAttempts - Maximum number of retry attempts (1 = no retries)
type ActivityRetryPolicy record {|
    int initialIntervalInSeconds = 1;
    decimal backoffCoefficient = 2.0;
    int maximumIntervalInSeconds?;
    int maximumAttempts = 1;
|};

// ---------------------------------------------------------------------------
// Activity retry policy types
// ---------------------------------------------------------------------------

# No automatic retry by the engine. Errors from the activity are returned
# directly to the caller. This is the default behaviour when no `retryPolicy`
# is specified. Note that an AI agent may still decide to call the activity
# again from its own reasoning — this policy only disables engine-driven
# retries.
public const NoRetry = ();

# No approval gate: the call runs as soon as it is made.
public const NoApproval = ();

# Automatic retry configuration. When the activity fails, it is automatically
# retried according to the configured backoff policy.
#
# + maxRetries - Maximum retry attempts (default: 3)
# + retryDelay - Initial delay in seconds before the first retry (default: 1.0)
# + retryBackoff - Multiplier applied to delay after each retry (default: 2.0)
# + maxRetryDelay - Cap on the delay between retries, in seconds
public type AutoRetry record {|
    int maxRetries = 3;
    decimal retryDelay = 1.0;
    decimal retryBackoff = 2.0;
    decimal maxRetryDelay?;
|};

# Automatic retries first; when they are spent, a person decides. Carries both an
# `AutoRetry` and a review's audience, so the review is raised only after the last
# automatic attempt fails. `maxRetries` is required here and forbidden on
# `ReviewTaskDefinition`, so a literal of either shape needs no cast.
#
# + maxRetries - Automatic attempts before the review is raised
public type RetryBeforeReview record {
    *AutoRetry;
    *ReviewTaskFields;
    int maxRetries;
};

# Failure behaviour of an activity call: fail at once (`NoRetry`), retry with backoff
# (`AutoRetry`), raise a review (`ReviewTaskDefinition`), or retry then review
# (`RetryBeforeReview`).
public type RetryPolicy AutoRetry|ReviewTaskDefinition|RetryBeforeReview|NoRetry;

# Whether a person must approve a call before it runs, and who. `NoApproval` runs the
# call directly; a `ReviewTaskDefinition` raises a `PRE_RUN` review first.
public type ApprovalPolicy ReviewTaskDefinition|NoApproval;

# Information about a registered workflow process.
#
# + name - The name of the registered process
# + activities - Array of activity names associated with this process
# + events - Array of event names (signals) this process can receive
type ProcessRegistration record {
    string name;
    string[] activities;
    string[] events;
};

# Information about all registered workflows.
# This is a map where keys are process names and values are their registration info.
type WorkflowRegistry map<ProcessRegistration>;

// ---------------------------------------------------------------------------
// HumanTask types
// ---------------------------------------------------------------------------

# Detail fields carried by a `HumanTaskTimeoutError`.
#
# + taskName - The `taskName` value passed to `awaitHumanTask`
# + taskWorkflowId - Temporal child workflow ID of the timed-out task instance
# + timedOutAfter - Configured deadline as an ISO-8601 duration (e.g. `"PT24H"`)
# + timedOutAt - ISO-8601 timestamp at which the timeout was recorded
public type HumanTaskTimeoutDetail record {|
    string taskName;
    string taskWorkflowId;
    string timedOutAfter;
    string timedOutAt;
|};

# Returned by `awaitHumanTask` when no human acts within the configured deadline.
# Catch the whole family with `on fail workflow:HumanTaskError e` and narrow with
# `if e is workflow:HumanTaskTimeoutError` to run compensation logic for a timeout.
public type HumanTaskTimeoutError distinct error<HumanTaskTimeoutDetail>;

# Detail fields carried by a `HumanTaskRejectedError`.
#
# + taskName - The `taskName` value passed to `awaitHumanTask`
# + taskWorkflowId - Temporal child workflow ID of the rejected task instance
# + reason - The reason submitted with the rejection
# + details - Structured data submitted with the rejection, or `()` if none was given
# + rejectedBy - The user who rejected the task, when the rejection recorded one
public type HumanTaskRejectedDetail record {|
    string taskName;
    string taskWorkflowId;
    string reason;
    map<json>? details = ();
    string? rejectedBy = ();
|};

# Returned by `awaitHumanTask` when the task is rejected instead of completed. The reason and any
# details submitted with the rejection are on the error detail, so a workflow can compensate on
# what the rejecting user said.
public type HumanTaskRejectedError distinct error<HumanTaskRejectedDetail>;

# Returned by `awaitHumanTask` when the task neither completed nor closed with a reason
# it can report — the task workflow failed, was terminated by an administrator, or the
# submitted value did not match the expected result type.
public type HumanTaskFailedError distinct error;

# Every failure `awaitHumanTask` can report: nobody acted in time
# (`HumanTaskTimeoutError`), someone rejected the task (`HumanTaskRejectedError`), or the
# task could not produce a result at all (`HumanTaskFailedError`).
public type HumanTaskError HumanTaskTimeoutError|HumanTaskRejectedError|HumanTaskFailedError;

# Who acted on a human task this workflow created, as recorded when the task closed.
#
# + taskId - Child workflow ID of the task instance
# + taskName - The qualified task name
# + completedBy - The user who completed or rejected it, when the completion recorded one
# + completedAt - ISO-8601 instant of the decision, when recorded
# + identitySource - Where that identity came from, when recorded
public type HumanTaskCompletion record {|
    string taskId;
    string taskName;
    string? completedBy = ();
    string? completedAt = ();
    string? identitySource = ();
    CompletionRole completedAs = AUDIENCE;
|};

# Who completed a task, relative to its definition: someone it was assigned to, or one of its
# administrators stepping in.
public enum CompletionRole {
    AUDIENCE = "audience",
    ADMINISTRATOR = "administrator"
}

// ---------------------------------------------------------------------------
// Review task types
// ---------------------------------------------------------------------------

# Detail fields carried by a `ReviewTimeoutError`.
#
# + taskName - The review's qualified task name
# + taskWorkflowId - Child workflow ID of the review instance
# + activityName - The activity under review
# + trigger - `PRE_RUN` for an approval gate, `ON_FAILURE` for a failed activity
# + timedOutAfter - Configured deadline as an ISO-8601 duration
# + timedOutAt - ISO-8601 timestamp at which the timeout was recorded
public type ReviewTimeoutDetail record {|
    string taskName;
    string taskWorkflowId;
    string activityName;
    string trigger;
    string timedOutAfter;
    string timedOutAt;
|};

# Returned when a review's deadline passes before a person decides. For a failed activity the
# original failure is the error's cause.
public type ReviewTimeoutError distinct error<ReviewTimeoutDetail>;

# Detail fields carried by a `ReviewRejectedError`.
#
# + taskName - The review's qualified task name
# + taskWorkflowId - Child workflow ID of the review instance
# + activityName - The activity under review
# + trigger - `PRE_RUN` for an approval gate, `ON_FAILURE` for a failed activity
# + feedback - The reviewer's note, when one was given
# + rejectedBy - The user who rejected it, when recorded
public type ReviewRejectedDetail record {|
    string taskName;
    string taskWorkflowId;
    string activityName;
    string trigger;
    string? feedback = ();
    string? rejectedBy = ();
|};

# Returned when a person rejects a review: a gated call is skipped, or a failed activity's
# failure stands. For a failed activity the original failure is the error's cause.
public type ReviewRejectedError distinct error<ReviewRejectedDetail>;

# Returned when a review ended without a usable decision — terminated, or its child failed.
public type ReviewFailedError distinct error;

# Every failure a review can report.
public type ReviewTaskError ReviewTimeoutError|ReviewRejectedError|ReviewFailedError;

# The decision a review reached, as this workflow saw it.
#
# + taskId - Child workflow ID of the review instance
# + taskName - The review's qualified task name
# + action - `proceed`, `proceed-with-input` or `reject`
# + feedback - The reviewer's note, when one was given
# + decidedBy - The user who decided, when recorded
# + decidedAt - ISO-8601 instant of the decision, when recorded
public type ReviewDecisionRecord record {|
    string taskId;
    string taskName;
    string action;
    string? feedback = ();
    string? decidedBy = ();
    string? decidedAt = ();
    CompletionRole completedAs = AUDIENCE;
|};

# A data-event turn a durable agent has accepted but not yet answered. Returned
# by `getPendingAgentEvents` so callers can rediscover in-flight event turns
# after a crash and fetch their answers via `DurableAgent.getDataResult` /
# `waitForDataResult`.
#
# + token - The turn's correlation token (as returned by `DurableAgent.sendData`)
# + eventName - The event channel the turn was sent on
public type PendingAgentEvent record {|
    string token;
    string eventName;
|};

// ---------------------------------------------------------------------------
// Child workflow types
// ---------------------------------------------------------------------------

# Where an instance is in its life, as `getStatus` reports it.
#
# + RUNNING - Executing, or waiting on a timer, an event or a task
# + SUSPENDED - Running but paused through the management API
# + COMPLETED - Closed with a result
# + FAILED - Closed with an error
# + CANCELED - Closed by a cancellation request
# + TERMINATED - Closed by a termination
# + TIMED_OUT - Closed because its execution timeout passed
public enum InstanceStatus {
    RUNNING,
    SUSPENDED,
    COMPLETED,
    FAILED,
    CANCELED,
    TERMINATED,
    TIMED_OUT
}

# The instance — or the turn — has not finished. Every non-blocking read answers this while the
# instance runs, and every bounded wait answers it when the bound passes first: `getResult`,
# `waitForResult`, `ctx->getChildWorkflowResult`, and a durable agent's `getResult`,
# `getDataResult`, `waitForResult` and `waitForDataResult`. Check back later, or wait.
public type WorkflowInProgressError distinct error;

# Detail fields carried by an `InstanceFailedError`.
#
# + instanceId - The instance
# + status - How it closed: `FAILED`, `CANCELED`, `TERMINATED` or `TIMED_OUT`
public type InstanceFailedDetail record {|
    string instanceId;
    InstanceStatus status;
|};

# Returned by a result read when the instance closed without a result: it failed, was cancelled
# or terminated, or timed out. The message carries the instance's own error message, when it
# had one; the detail says how it closed.
public type InstanceFailedError distinct error<InstanceFailedDetail>;

# Detail fields carried by an `InstanceNotFoundError`.
#
# + instanceId - The id nothing holds
public type InstanceNotFoundDetail record {|
    string instanceId;
|};

# Returned by a read when no instance holds the id — or, from a durable agent's method, when the
# instance is not one of that agent's.
public type InstanceNotFoundError distinct error<InstanceNotFoundDetail>;

# What a start does when its caller-chosen instance id is held by a RUNNING instance.
#
# + FAIL - Refuse with an `InstanceAlreadyExistsError` (the default)
# + USE_EXISTING - Return the running instance's id and leave it as it is; the new input is not
#                  delivered. This is the idempotent submit: a retried request joins the first
public enum RunningInstancePolicy {
    FAIL,
    USE_EXISTING
}

# What a start does when its caller-chosen instance id was held by an instance that has CLOSED.
#
# + ALLOW_DUPLICATE - Start a new run under the same id, whatever the old outcome (the default)
# + ALLOW_DUPLICATE_FAILED_ONLY - Start a new run only if the old one failed, was cancelled,
#                                 terminated or timed out; refuse after a completed one
# + REJECT_DUPLICATE - Refuse while the engine still retains the closed instance. Retention is
#                      finite (a namespace setting, days by default), so this is not a permanent
#                      guard against processing a business key twice — keep such a guard in the domain
public enum ClosedInstancePolicy {
    ALLOW_DUPLICATE,
    ALLOW_DUPLICATE_FAILED_ONLY,
    REJECT_DUPLICATE
}

# Detail fields carried by an `InstanceAlreadyExistsError`.
#
# + instanceId - The id the start asked for
# + status - Status of the instance holding it — `RUNNING`, or a closed status such as
#            `COMPLETED` or `FAILED`
public type InstanceAlreadyExistsDetail record {|
    string instanceId;
    string status;
|};

# Returned by `runWithId` (and `DurableAgent.runWithId`) when the id is held by an instance the
# policies did not let the start replace or join.
public type InstanceAlreadyExistsError distinct error<InstanceAlreadyExistsDetail>;

# Any JSON object.
public type JsonObject map<json>;

# Who may answer a human decision, and how it reads. Shared by a workflow's human task, a
# durable agent's task capability, and the review a gated activity raises.
#
# At least one of `userRoles` and `users` must name someone. `userRoles` is required so that a
# plain `AutoRetry` literal is never mistaken for a review; write `userRoles: ()` when the
# audience is given by `users` alone.
#
# + userRoles - Role(s) permitted to answer this decision, or `()` when only `users` may
# + users - User id(s) permitted to answer it, whatever their roles
# + excludedUsers - User id(s) that may not answer it, whatever their roles
# + excludedRoles - Role(s) that may not answer it
# + administratorRoles - Role(s) that administer it: they see it, may reassign it, move its deadline,
#                        fail it, or complete it — a completion by them is recorded as an administrator's
# + administratorUsers - User id(s) that administer it, whatever their roles
# + title - Short summary shown in the inbox. Defaults to the task name
# + description - Additional context shown with the form or decision
# + timeout - Maximum time to wait. Omit to wait indefinitely
type ReviewTaskFields record {
    string|[string, string...]? userRoles;
    string|[string, string...] users?;
    string|[string, string...] excludedUsers?;
    string|[string, string...] excludedRoles?;
    string|[string, string...] administratorRoles?;
    string|[string, string...] administratorUsers?;
    string? title = ();
    string? description = ();
    Duration? timeout = ();
};

# A review's whole definition: its audience and wording. A human task adds the shapes it
# is checked against — see `HumanTaskDefinition`. `maxRetries` is forbidden so a
# `RetryBeforeReview` literal is never mistaken for a plain review.
#
# + maxRetries - Never present; retries belong to `RetryBeforeReview`
public type ReviewTaskDefinition record {
    *ReviewTaskFields;
    never maxRetries?;
};

# A human task: who may answer it and how it reads, plus the shapes it takes and returns.
#
# The input supplied to the task is checked against `taskInputType` before the task is
# created, whether a workflow passes it to `awaitHumanTask` or an agent supplies it.
#
# + taskInputType - Shape of the input shown to the decider
# + resultType - Shape of the answer. A workflow states this as `awaitHumanTask`'s `T`
#                instead; an agent declares it here
public type HumanTaskDefinition record {
    *ReviewTaskDefinition;
    typedesc<map<json>> taskInputType = JsonObject;
    typedesc<anydata> resultType = anydata;
};

# How a `Context.callActivity` invocation behaves, passed as an included record
# parameter. Deliberately an OPEN record so a future behaviour
# option — an approval gate, a heartbeat policy, a per-call timeout — is a new field
# here rather than a new parameter, and tooling derives its forms from this record.
# The step identity (`stepId`) is NOT here: it is workflow mechanics, not invocation
# behaviour, and stays a function parameter on every context operation.
#
# + approvalPolicy - Whether a person approves the call before it runs, and who
# + retryPolicy - Failure behaviour: `NoRetry` (fail the workflow), `AutoRetry`
#                 (durable backoff retries), a `ReviewTaskDefinition` (raise a review
#                 on failure so a person decides to rerun, rerun with edited input, or
#                 fail), or `RetryBeforeReview` (retry, then review)
public type CallActivityOptions record {
    ApprovalPolicy approvalPolicy = NoApproval;
    RetryPolicy retryPolicy = NoRetry;
};

# A time duration, structurally identical to `time:Duration`. Declared in this module so
# timeout fields render as first-class workflow forms without a cross-module type reference;
# `time:Duration` values remain assignable.
public type Duration record {|
    # The duration in years
    int years = 0;
    # The duration in months
    int months = 0;
    # The duration in weeks
    int weeks = 0;
    # The duration in days
    int days = 0;
    # The duration in hours
    int hours = 0;
    # The duration in minutes
    int minutes = 0;
    # The duration in seconds
    decimal seconds = 0.0;
|};
