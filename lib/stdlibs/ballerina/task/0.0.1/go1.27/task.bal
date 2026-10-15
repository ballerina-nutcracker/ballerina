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

import ballerina/time;
import ballerina/uuid;

# Represents the error type of the `ballerina/task` module. This error type represents any error that can occur during
# the execution of the task APIs.
public type Error distinct error;

# Represents the task service that provides functionality to manage and execute scheduled tasks.
public type Service distinct service object {
    public isolated function execute() returns error?;
};

# Represents the configuration required to connect to a database related to task coordination.
public type DatabaseConfig MysqlConfig|PostgresqlConfig;

# Represents the configuration required to connect to a database related to task coordination.
#
# + host - The hostname of the database server
# + user - The username for the database connection
# + password - The password for the database connection
# + port - The port number of the database server
# + database - The name of the database to connect to
public type MysqlConfig record {
    string host = "localhost";
    string? user = ();
    string? password = ();
    int port = 3306;
    string? database = ();
};

# Represents the configuration required to connect to a database related to task coordination.
#
# + host - The hostname of the database server
# + user - The username for the database connection
# + password - The password for the database connection
# + port - The port number of the database server
# + database - The name of the database to connect to
public type PostgresqlConfig record {
    string host = "localhost";
    string? user = ();
    string? password = ();
    int port = 5432;
    string? database = ();
};

# Represents the configuration required for task coordination.
#
# + databaseConfig - The database configuration for task coordination
# + livenessCheckInterval - The interval (in seconds) to check the liveness of the job. Default is 30 seconds.
# + taskId - Unique identifier for the current task
# + groupId - The identifier for the group of tasks. This is used to identify the group of tasks that are
#             coordinating the task. It is recommended to use a unique identifier for each group of tasks.
# + heartbeatFrequency - The interval (in seconds) for the node to update its heartbeat. Default is one second.
public type WarmBackupConfig record {
    DatabaseConfig databaseConfig = defaultDatabaseConfig();
    int livenessCheckInterval = 30;
    string taskId;
    string groupId;
    int heartbeatFrequency = 1;
};

isolated function defaultDatabaseConfig() returns MysqlConfig => {};

# Worker count for the global scheduler
public int globalSchedulerWorkerCount = 5;

# Waiting time for the global scheduler
public time:Seconds globalSchedulerWaitingTime = 5;

# Listener configuration.
#
# + trigger - The trigger configuration for the listener
# + warmBackupConfig - The configuration related to task coordination
public type ListenerConfiguration record {
    TriggerConfiguration trigger;
    WarmBackupConfig? warmBackupConfig = ();
};

# Recurring schedule configuration.
#
# + interval - The duration of the trigger (in seconds), which is used to run the job frequently
# + maxCount - The maximum number of trigger counts. If set to -1, job will run indefinitely
# + startTime - The trigger start time in Ballerina `time:Civil`. If it is not provided, a trigger will
#               start immediately
# + endTime - The trigger end time in Ballerina `time:Civil`
# + taskPolicy - The policy, which is used to handle the error and will be waiting during the trigger time
# + retryConfig - The retry configurations for job executions
public type TriggerConfiguration record {|
    decimal interval;
    int maxCount = -1;
    time:Civil startTime?;
    time:Civil endTime?;
    TaskPolicy taskPolicy = {
        errorPolicy: LOG_AND_TERMINATE,
        waitingPolicy: WAIT
    };
    RetryConfiguration? retryConfig = ();
|};

# Retry configuration for job execution.
#
# + maxAttempts - Maximum number of retry attempts
# + retryInterval - Initial retry interval
# + backoffStrategy - The strategy to use for retrying the job execution
# + maxInterval - Maximum amount of time to wait between retries
public type RetryConfiguration record {|
    int maxAttempts;
    int retryInterval;
    RetryStrategy backoffStrategy = FIXED;
    int maxInterval?;
|};

# Supported retry strategies for job execution.
public enum RetryStrategy {
    FIXED,
    EXPONENTIAL
};

# A read-only record consisting of a unique identifier for a created job.
public type JobId readonly & record {|
    int id;
|};

# The Ballerina Job object provides the abstraction for a job instance, which schedules to execute periodically.
public type Job object {

    # Executes by the Scheduler when the scheduled trigger fires.
    public function execute();
};

# Policies related to a trigger.
#
# + errorPolicy - The policy to follow when there is an error in Job execution
# + waitingPolicy - The policy to follow when the next task is triggering while the previous job is still
#                   being processing
public type TaskPolicy record {|
    ErrorPolicy errorPolicy = LOG_AND_TERMINATE;
    WaitingPolicy waitingPolicy = WAIT;
|};

# Possible options for the `ErrorPolicy`.
public enum ErrorPolicy {
    LOG_AND_TERMINATE,
    LOG_AND_CONTINUE,
    TERMINATE,
    CONTINUE
}

# Possible options for the `WaitingPolicy`.
public enum WaitingPolicy {
    WAIT,
    IGNORE,
    LOG_AND_IGNORE
}

# Gets time in milliseconds of the given `time:Civil`.
#
# + time - The Ballerina `time:Civil`
# + return - Time in milliseconds or else `task:Error` if the process failed due to any reason
public isolated function getTimeInMillies(time:Civil time) returns int|Error {
    if time[UTC_OFF_SET] == () {
        time:ZoneOffset zoneOffset = {hours: 0, minutes: 0};
        time.utcOffset = zoneOffset;
    }
    time:Utc|time:Error utc = time:utcFromCivil(time);
    if utc is time:Utc {
        if utc > time:utcNow() {
            return <int>decimal:round((<decimal>utc[0] + utc[1]) * 1000);
        }
        return error Error(
                    string `Invalid time: ${time.toString()}. Scheduled time should be greater than the current time`);
    } else {
        return error Error(string `Couldn't convert given time to milli seconds: ${utc.message()}.`);
    }
}

const string UTC_OFF_SET = "utcOffset";

# Configure the scheduler worker pool.
# ```ballerina
# check task:configureWorkerPool(4, 7);
# ```
#
# + workerCount - Specifies the number of workers that are available for the concurrent execution of jobs.
#                 It should be a positive integer. The recommendation is to set a value less than 10. Default sets to 5.
# + waitingTime - The number of seconds as a decimal the scheduler will tolerate a trigger to pass its next-fire-time
#                 before being considered as `ignored the trigger`
# + return - A `task:Error` if the process failed due to any reason or else ()
public isolated function configureWorkerPool(int workerCount = 5, time:Seconds waitingTime = 5)
                                returns Error? {
    return toError(configureThread(workerCount, <int>(waitingTime * <decimal>1000.0)));
}

# Schedule the given `task:Job` for the given time. Once scheduled, it will return a job ID, which can be used to manage
# the job.
# ```ballerina
# time:Utc newTime = time:utcAddSeconds(time:utcNow(), 3);
# time:Civil time = time:utcToCivil(newTime);
# task:JobId jobId = check task:scheduleOneTimeJob(new Job(), time);
# ```
#
# + triggerTime - The specific time in Ballerina `time:Civil` to trigger only one time
# + job - Ballerina job, which is to be executed during the trigger
# + return - A `task:JobId` or else a `task:Error` if the process failed due to any reason
public isolated function scheduleOneTimeJob(Job job, time:Civil triggerTime) returns JobId|Error {
    int|string result = scheduleJob(job, check getTimeInMillies(triggerTime));
    return toJobId(result);
}

# Schedule the recurring `task:Job` according to the given duration. Once scheduled, it will return the job ID, which
# can be used to manage the job.
# ```ballerina
# task:JobId jobId = check task:scheduleJobRecurByFrequency(new Job(), 3);
# ```
#
# + job - Ballerina job, which is to be executed by the scheduler
# + interval - The duration of the trigger (in seconds), which is used to run the job frequently
# + maxCount - The maximum number of trigger counts. If set to -1, job will run indefinitely
# + startTime - The trigger start time in Ballerina `time:Civil`. If it is not provided, a trigger will
#               start immediately
# + endTime - The trigger end time in Ballerina `time:Civil`
# + taskPolicy -  The policy, which is used to handle the error and will be waiting during the trigger time
# + return - A `task:JobId` or else a `task:Error` if the process failed due to any reason
public isolated function scheduleJobRecurByFrequency(Job job, decimal interval, int maxCount = -1,
        time:Civil? startTime = (), time:Civil? endTime = (), TaskPolicy taskPolicy = {})
    returns JobId|Error {
    if maxCount != -1 && maxCount < 1 {
        return error Error("The maxCount should be a positive integer.");
    }
    int? sTime = ();
    int? eTime = ();
    if startTime != () {
        sTime = check getTimeInMillies(startTime);
    }
    if endTime != () {
        eTime = check getTimeInMillies(endTime);
    }
    int|string result = scheduleIntervalJob(job, interval, maxCount, sTime, eTime, taskPolicy);
    return toJobId(result);
}

# Unschedule the `task:Job`, which is associated with the given job ID. If no job is running in the scheduler,
# the scheduler will be shut down automatically.
# ```ballerina
# check task:unscheduleJob(jobId);
# ```
#
# + jobId - The ID of the job as a `task:JobId`, which needs to be unscheduled
# + return - A `task:Error` if the process failed due to any reason or else ()
public isolated function unscheduleJob(JobId jobId) returns Error? {
    return toError(externUnscheduleJob(jobId.id));
}

# Pauses all the jobs.
# ```ballerina
# check task:pauseAllJobs();
# ```
#
# + return - A `task:Error` if an error occurred while pausing or else ()
public isolated function pauseAllJobs() returns Error? {
    return toError(externPauseAllJobs());
}

# Resumes all the jobs.
# ```ballerina
# check task:resumeAllJobs();
# ```
#
# + return - A `task:Error` when an error occurred while resuming or else ()
public isolated function resumeAllJobs() returns Error? {
    return toError(externResumeAllJobs());
}

# Pauses the particular job.
# ```ballerina
# check task:pauseJob(jobId);
# ```
#
# + jobId - The ID of the job as a `task:JobId`, which needs to be paused
# + return - A `task:Error` if an error occurred while pausing a job or else ()
public isolated function pauseJob(JobId jobId) returns Error? {
    return toError(externPauseJob(jobId.id));
}

# Resumes the particular job.
# ```ballerina
# check task:resumeJob(jobId);
# ```
#
# + jobId - The ID of the job as a `task:JobId`, which needs to be resumed
# + return - A `task:Error` when an error occurred while resuming a job or else ()
public isolated function resumeJob(JobId jobId) returns Error? {
    return toError(externResumeJob(jobId.id));
}

# Gets all the running jobs.
# ```ballerina
# task:JobId[] result = task:getRunningJobs();
# ```
#
# + return - IDs of all the running jobs as an array
public isolated function getRunningJobs() returns JobId[] {
    JobId[] jobIds = [];
    foreach int id in externGetRunningJobs() {
        jobIds.push({id: id});
    }
    return jobIds;
}

isolated function toJobId(int|string result) returns JobId|Error {
    if result is string {
        return error Error(result);
    }
    JobId jobId = {id: result};
    return jobId;
}

isolated function toError(string? message) returns Error? {
    if message is string {
        return error Error(message);
    }
    return ();
}

isolated function scheduleJob(Job job, int triggerTime) returns int|string = external;

isolated function configureThread(int workerCount, int waitingTime) returns string? = external;

isolated function scheduleIntervalJob(Job job, decimal interval, int maxCount, int? startTime, int? endTime,
        TaskPolicy taskPolicy) returns int|string = external;

isolated function externUnscheduleJob(int id) returns string? = external;

isolated function externPauseAllJobs() returns string? = external;

isolated function externResumeAllJobs() returns string? = external;

isolated function externPauseJob(int id) returns string? = external;

isolated function externResumeJob(int id) returns string? = external;

isolated function externGetRunningJobs() returns int[] = external;

# Listener for task scheduling.
public isolated class Listener {

    # Initializes the task 'listener.
    #
    # + config - The 'listener configuration
    public isolated function init(*ListenerConfiguration config) returns Error? {
        self.initListener(config);
    }

    # Attaches a service to the 'listener.
    #
    # + s - The service to be attached
    # + name - The service name
    # + return - An error if the service cannot be attached
    public isolated function attach(Service s, string[]|string? name = ()) returns Error? {
        string serviceId = name is string ? name : (name is string[] ? name[0] : uuid:createRandomUuid());
        self.attachService(s, serviceId);
    }

    # Detaches a service from the 'listener.
    #
    # + s - The service to be detached
    # + return - An error if the service cannot be detached
    public isolated function detach(Service s) returns Error? {
        return toError(self.detachService(s));
    }

    # Starts the 'listener.
    #
    # + return - An error if the 'listener fails to start
    public isolated function 'start() returns Error? {
        return toError(self.startListener());
    }

    # Stops the 'listener gracefully.
    #
    # + return - An error if the 'listener fails to stop
    public isolated function gracefulStop() returns Error? {
        return toError(self.stopListener());
    }

    # Stops the 'listener immediately.
    #
    # + return - An error if the 'listener fails to stop
    public isolated function immediateStop() returns Error? {
        return toError(self.stopListener());
    }

    isolated function initListener(ListenerConfiguration config) = external;

    isolated function attachService(Service s, string serviceId) = external;

    isolated function detachService(Service s) returns string? = external;

    isolated function startListener() returns string? = external;

    isolated function stopListener() returns string? = external;
}
