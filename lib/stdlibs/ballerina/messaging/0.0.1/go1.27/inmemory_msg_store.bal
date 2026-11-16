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

import ballerina/log;
import ballerina/uuid;

type InMemoryMessage record {|
    *Message;
    boolean inFlight = false;
|};

# Represents an in-memory message store.
public isolated client class InMemoryMessageStore {
    *Store;

    private InMemoryMessage[] messages;

    # Initializes a new instance of the InMemoryStore.
    public isolated function init() {
        self.messages = [];
    }

    # Stores a message in the message store.
    #
    # + payload - The message payload to be stored
    isolated remote function store(anydata payload) {
        lock {
            string id = uuid:createType1AsString();
            self.messages.push({id, payload: payload.clone()});
        }
    }

    # Retrieves the top message from the message store.
    #
    # + return - The retrieved message, or `()` if the store is empty
    isolated remote function retrieve() returns Message? {
        lock {
            if self.messages.length() == 0 {
                return;
            }
            foreach InMemoryMessage message in self.messages {
                if !message.inFlight {
                    message.inFlight = true;
                    return {id: message.id, payload: message.payload.clone()};
                }
            }
            return;
        }
    }

    # Acknowledges the processing of a message.
    #
    # + id - The unique identifier of the message to acknowledge
    # + success - Indicates whether the message was processed successfully
    # + return - An error if the acknowledgment could not be processed, or `()`
    isolated remote function acknowledge(string id, boolean success = true) returns error? {
        lock {
            InMemoryMessage[] targetMessage = from InMemoryMessage message in self.messages
                where message.id == id && message.inFlight
                limit 1
                select message;
            if targetMessage.length() == 0 {
                return error("Message with the given ID not found or not in flight", id = id);
            }

            InMemoryMessage message = targetMessage[0];
            int? targetIndex = self.messages.indexOf(message);
            if targetIndex is () {
                // This should not happen as we already checked the existence of the message
                return error("Message with the given ID not found in the store", id = id);
            }
            if success {
                _ = self.messages.remove(targetIndex);
            } else {
                log:printDebug("acknowledged with failure, message is kept in the store", msgId = id);
                message.inFlight = false;
            }
        }
    }
}
