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

import ballerina/mime;

// Builds a received email:Message for the native receive functions. headerEntries alternates
// names and values in message order, so a repeated header keeps its last value.
isolated function newReceivedMessage(string[] to, string[] cc, string[] bcc, string[] replyTo, string subject,
        string body, string? htmlBody, string contentType, string[] headerEntries, string 'from, string sender,
        mime:Entity[] attachments) returns Message {
    Message message = {
        to: addressField(to),
        cc: addressField(cc),
        bcc: addressField(bcc),
        replyTo: addressField(replyTo),
        subject,
        body,
        contentType,
        'from,
        sender
    };
    if htmlBody is string {
        message.htmlBody = htmlBody;
    }
    if headerEntries.length() > 0 {
        map<string> headers = {};
        foreach int i in 0 ..< headerEntries.length() / 2 {
            headers[headerEntries[2 * i]] = headerEntries[2 * i + 1];
        }
        message.headers = headers;
    }
    if attachments.length() > 0 {
        message.attachments = attachments;
    }
    return message;
}

// A single address is a string and any other number a list, as jBallerina returns them.
isolated function addressField(string[] addresses) returns string|string[] =>
    addresses.length() == 1 ? addresses[0] : addresses;
