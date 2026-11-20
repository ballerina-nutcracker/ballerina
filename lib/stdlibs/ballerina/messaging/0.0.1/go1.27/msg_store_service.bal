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

# This service object defines the contract for processing messages from a message store.
public type StoreService distinct isolated service object {

    # This function is called when a new message is received from the message store.
    #
    # + payload - The message payload to be processed
    # + return - An error if the message could not be processed, or a nil value
    isolated remote function onMessage(anydata payload) returns error?;
};
