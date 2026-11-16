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

import ballerina/lang.runtime;
import ballerina/log;
import ballerina/task;

# Represents the default error type.
public type Error distinct error;

# Represents the message store listener configuration,
public type StoreListenerConfiguration record {|
    # The interval in seconds at which the listener polls for new messages
    decimal pollingInterval = 1;
    # The maximum number of retries for processing a message. 
    # If set to 0, the message will not be retried
    int maxRetries = 3;
    # The interval in seconds between retries for processing a message
    decimal retryInterval = 1;
    # If true, the message will be acknowledged with failure after the maximum number of retries is reached.
    # Else the message will be acknowledged with success.
    boolean ackWithFailureAfterMaxRetries = true;
    # An optional message store to store messages that could not be processed after the maximum 
    # number of retries. On successful storage, the message will be acknowledged with success
    Store deadLetterStore?;
|};

# Represents a message store listener that polls messages from a message store and processes them.
public isolated class StoreListener {

    private final ListenerState state;
    private final readonly & PollConfig config;

    # Initializes a new instance of Message Store Listener.
    #
    # + messageStore - The message store to retrieve messages from
    # + config - The configuration for the message store listener
    # + return - An error if the listener could not be initialized, or `()`
    public isolated function init(Store messageStore, *StoreListenerConfiguration config) returns Error? {
        self.state = {messageStore, deadLetterStore: config?.deadLetterStore};
        self.config = {
            pollingInterval: config.pollingInterval,
            maxRetries: config.maxRetries,
            retryInterval: config.retryInterval,
            ackWithFailureAfterMaxRetries: config.ackWithFailureAfterMaxRetries
        };
        if config.maxRetries < 0 {
            return error Error("maxRetries must be greater than or equal to zero");
        }
        if config.pollingInterval <= 0d {
            return error Error("pollingInterval must be greater than zero");
        }
        if config.retryInterval <= 0d {
            return error Error("retryInterval must be greater than zero");
        }
    }

    # Attaches a message store service to the listener. Only one service can be attached to this 
    # listener.
    #
    # + msgStoreService - The message store service to attach
    # + path - The path is not relevant for this listener. Only allowing a nil value
    # + return - An error if the service could not be attached, or a nil value
    public isolated function attach(StoreService msgStoreService, () path = ()) returns Error? {
        _ = path;
        lock {
            if self.state.messageStoreService is StoreService {
                return error Error("service is already attached. Only one service can be " +
                    "attached to the message store listener");
            }
            self.state.messageStoreService = msgStoreService;
        }
    }

    # Detaches the message store service from the listener.
    #
    # + msgStoreService - The message store service to detach
    # + return - An error if the service could not be detached, or a nil value
    public isolated function detach(StoreService msgStoreService) returns Error? {
        lock {
            StoreService? currentService = self.state.messageStoreService;
            if currentService is () {
                return error Error("no service is attached");
            }
            if currentService !== msgStoreService {
                return error Error("the provided service is not attached to the listener");
            }
            error? stopResult = self.stopPolling();
            if stopResult is error {
                return error Error("failed to detach the service", cause = stopResult);
            }
            self.state.messageStoreService = ();
        }
    }

    # Starts the message store listener to poll and process messages.
    #
    # + return - An error if the listener could not be started, or a nil value
    public isolated function 'start() returns Error? {
        lock {
            StoreService? currentService = self.state.messageStoreService;
            if currentService is () || self.state.pollJobId !is () {
                return;
            }

            Cancellation cancellation = new;
            PollAndProcessMessages pollTask = new (self.state.messageStore, currentService, self.config,
                self.state.deadLetterStore, cancellation);
            task:JobId|error pollJob = task:scheduleJobRecurByFrequency(pollTask, self.config.pollingInterval);
            if pollJob is error {
                return error Error("failed to start message store listener", cause = pollJob);
            }
            self.state.pollJobId = pollJob;
            self.state.cancellation = cancellation;
        }
    }

    # Gracefully stops the message store listener by waiting for any ongoing processing to 
    # complete before stopping. This is not implemented yet, and currently this will call 
    # immediateStop.
    #
    # + return - An error if the listener could not be stopped, or a nil value
    public isolated function gracefulStop() returns Error? {
        return self.immediateStop();
    }

    # Immediately stops the message store listener without waiting for any ongoing processing 
    # to complete.
    #
    # + return - An error if the listener could not be stopped, or `()`.
    public isolated function immediateStop() returns Error? {
        error? stopResult = self.stopPolling();
        if stopResult is error {
            return error Error("failed to stop message store listener", cause = stopResult);
        }
    }

    isolated function stopPolling() returns error? {
        lock {
            task:JobId? jobId = self.state.pollJobId;
            if jobId is () {
                return;
            }
            check task:unscheduleJob(jobId);
            self.state.pollJobId = ();
            Cancellation? cancellation = self.state.cancellation;
            if cancellation is Cancellation {
                cancellation.cancel();
            }
            self.state.cancellation = ();
        }
    }

}

type ListenerState record {|
    Store messageStore;
    Store? deadLetterStore;
    StoreService? messageStoreService = ();
    task:JobId? pollJobId = ();
    Cancellation? cancellation = ();
|};

isolated class Cancellation {
    private boolean cancelled = false;

    // Explicit init works around https://github.com/ballerina-nutcracker/ballerina/issues/1076
    isolated function init() {
    }

    isolated function cancel() {
        lock {
            self.cancelled = true;
        }
    }

    isolated function isCancelled() returns boolean {
        lock {
            return self.cancelled;
        }
    }
}

type PollConfig record {|
    decimal pollingInterval;
    int maxRetries;
    decimal retryInterval;
    boolean ackWithFailureAfterMaxRetries;
|};

class PollAndProcessMessages {
    *task:Job;

    private final Store messageStore;
    private final StoreService messageStoreService;
    private final readonly & PollConfig config;
    private final Store? deadLetterStore;
    private final Cancellation cancellation;

    public isolated function init(Store messageStore, StoreService messageStoreService,
            readonly & PollConfig config, Store? deadLetterStore, Cancellation cancellation) {
        self.messageStore = messageStore;
        self.messageStoreService = messageStoreService;
        self.config = config;
        self.deadLetterStore = deadLetterStore;
        self.cancellation = cancellation;
    }

    public isolated function ackMessage(string id, boolean success = true) {
        error? result = self.messageStore->acknowledge(id, success);
        if result is error {
            log:printError("failed to acknowledge message", result, msgId = id);
        }
    }

    public isolated function execute() {
        Message|error? message = self.messageStore->retrieve();
        if message is error {
            log:printError("error occurred while polling for the message",  message);
            return;
        }
        if message is () {
            return;
        }

        anydata payload = message.payload;
        string id = message.id;

        error? result = trap invokeOnMessage(self.messageStoreService, payload.clone());
        if result is () {
            self.ackMessage(id);
            return;
        }
        log:printError("error occurred while processing message", result, msgId = id);

        if self.config.maxRetries > 0 {
            foreach int attempt in 1 ... self.config.maxRetries {
                runtime:sleep(self.config.retryInterval);
                if self.cancellation.isCancelled() {
                    // Released so the store can deliver it again once the listener restarts.
                    self.ackMessage(id, false);
                    return;
                }
                error? retryResult = trap invokeOnMessage(self.messageStoreService, payload.clone());
                if retryResult is error {
                    log:printError("error processing message on retry", retryResult, retryAttempt = attempt, msgId = id);
                } else {
                    log:printDebug("message processed successfully on retry", retryAttempt = attempt, msgId = id);
                    self.ackMessage(id);
                    return;
                }
            }
        }
        Store? dls = self.deadLetterStore;
        if dls is Store {
            error? dlsResult = dls->store(payload);
            if dlsResult is error {
                log:printError("failed to store message in dead letter store", dlsResult, msgId = id);
            } else {
                log:printDebug("message stored in dead letter store after max retries", msgId = id);
                self.ackMessage(id);
                return;
            }
        }

        if self.config.ackWithFailureAfterMaxRetries {
            log:printDebug("max retries reached, acknowledging message with failure", msgId = id);
        } else {
            log:printDebug("max retries reached, acknowledging message with success", msgId = id);
        }
        self.ackMessage(id, !self.config.ackWithFailureAfterMaxRetries);
    }
}

isolated function invokeOnMessage(StoreService svc, anydata payload) returns error? = external;
