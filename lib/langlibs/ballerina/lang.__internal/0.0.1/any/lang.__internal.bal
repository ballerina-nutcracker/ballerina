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

public isolated function querySort(any[] sortKeyRows, any[] sortDirections, any[] rowIndices, any[] payloadRows) = external;

public isolated function queryGroup(any[] rows, any[] keyRows, any[] scalarFlags) returns any[] = external;

public isolated function queryCollect(any[] rows, int slotCount, any[] flattenFlags) returns any[] = external;

public isolated function escapeXMLContent(string|boolean|int|float|decimal value) returns string = external;

public isolated function escapeXMLAttribute(string|boolean|int|float|decimal value) returns string = external;

# Creates a fresh closed one-shot latch. Strands that wait on it are held until
# it is opened, and it can never be closed again.
public isolated function createLatch() returns handle = external;

# Returns only after the latch opens, yielding cooperatively while it is closed.
public isolated function waitOnLatch(handle latch) = external;

# Opens the latch, releasing every waiting strand. A latch may be opened once.
public isolated function openLatch(handle latch) = external;

# Creates a message one worker sends to another.
public isolated function createWorkerMessage() returns handle = external;

# Stores a clone of the value sent as the message.
public isolated function setWorkerMessageValue(handle message, any|error value) = external;

# Returns the value of the message once its sender stored it, or the sender's
# failure if it terminates first.
public isolated function getWorkerMessageValue(handle message, future<any|error> sender) returns any|error = external;

# Returns the values of the messages once every sender stored its message, or
# the failure of a sender that terminates first.
public isolated function getWorkerMessageValues(handle[] messages, future<any|error>[] senders)
        returns (any|error)[]|error = external;

# Returns once the receiver took the message, or the receiver's failure if it
# terminates first.
public isolated function waitWorkerMessageReceived(handle message, future<any|error> receiver) returns error? = external;

# Returns once each receiver took every message sent to it. Of the receivers
# that terminate leaving a message unreceived, a panic is raised first, else
# the first error is returned.
public isolated function flushWorkerMessages(handle[][] messages, future<any|error>[] receivers) returns error? = external;

# Returns once every message was received; panics with the termination value
# of a receiver that terminates leaving one unreceived.
public isolated function awaitWorkerMessageDelivery(handle[] messages, future<any|error>[] receivers) = external;
