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

import ballerina/mime;

# Represents an SMTP Client, which interacts with an SMTP Server.
public isolated client class SmtpClient {

    # Gets invoked during the `email:SmtpClient` initialization.
    #
    # + host - Host of the SMTP Client
    # + username - Username of the SMTP Client if authentication is required
    # + password - Password of the SMTP Client if basic authentication is used
    # + clientConfig - Configurations for SMTP Client
    # + return - An `email:Error` if failed to initialize or else `()`
    public isolated function init(string host, string? username = (), string? password = (),
            *SmtpConfiguration clientConfig) returns Error? {
        if username is string && password is string {
            return toEmailError(initSmtpClientEndpoint(self, host, username, password, clientConfig));
        }
        if username is () && password is () {
            return toEmailError(initNoAuthSmtpClientEndpoint(self, host, clientConfig));
        }
        return error Error("Invalid configuration: provide 'username'+'password' for basic auth, " +
                           "'username'+'auth' for OAuth2, or neither for unauthenticated mode.");
    }

    # Sends an email message.
    # ```ballerina
    # email:Error? response = smtpClient->sendMessage(email);
    # ```
    #
    # + email - An `email:Message` message, which needs to be sent to the recipient
    # + return - An `email:Error` if failed to send the message to the recipient or else `()`
    remote isolated function sendMessage(Message email) returns Error? {
        if email.contentType is string && !self.containsType(email?.contentType, "text") {
            return error Error("Content type of the email should be text.");
        }
        return toEmailError(send(self, email));
    }

    # Sends an email message with optional parameters.
    # ```ballerina
    # email:Error? response = smtpClient->send(toAddress, subject, fromAddress,
    #   emailBody, sender="eve@abc.com");
    # ```
    #
    # + to - TO address list
    # + subject - Subject of email
    # + from - From address
    # + body - Text body of the email
    # + options - Optional parameters of the email
    # + return - An `email:Error` if failed to send the message to the recipient or else `()`
    remote isolated function send(string|string[] to, string subject, string 'from, string body, *Options options)
            returns Error? {
        Message email = {
            to: to,
            subject: subject,
            'from: 'from,
            body: body
        };
        string? htmlBody = options?.htmlBody;
        if htmlBody is string {
            email.htmlBody = htmlBody;
        }
        string? contentType = options?.contentType;
        if contentType is string {
            email.contentType = contentType;
        }
        map<string>? headers = options?.headers;
        if headers is map<string> {
            email.headers = headers;
        }
        string|string[]? cc = options?.cc;
        if cc !is () {
            email.cc = cc;
        }
        string|string[]? bcc = options?.bcc;
        if bcc !is () {
            email.bcc = bcc;
        }
        string|string[]? replyTo = options?.replyTo;
        if replyTo !is () {
            email.replyTo = replyTo;
        }
        string? sender = options?.sender;
        if sender is string {
            email.sender = sender;
        }
        mime:Entity|Attachment|(mime:Entity|Attachment)[]? attachments = options?.attachments;
        if attachments !is () {
            email.attachments = attachments;
        }
        return self->sendMessage(email);
    }

    private isolated function containsType(string? contentType, string typeString) returns boolean {
        if contentType is string {
            return contentType.toLowerAscii().indexOf(typeString) is int;
        }
        return false;
    }
}

# Configuration of the SMTP Endpoint.
#
# + port - Port number of the SMTP server
# + security - Type of security channel
# + secureSocket - Secure socket configuration
public type SmtpConfiguration record {|
    int port = 465;
    Security security = SSL;
    SecureSocket secureSocket?;
|};

isolated function toEmailError(error? err) returns Error? {
    if err is error {
        return error Error(err.message(), err.cause());
    }
    return ();
}

isolated function initSmtpClientEndpoint(SmtpClient clientEndpoint, string host, string username, string password,
        SmtpConfiguration config) returns error? = external;

isolated function initNoAuthSmtpClientEndpoint(SmtpClient clientEndpoint, string host, SmtpConfiguration config)
        returns error? = external;

isolated function send(SmtpClient clientEndpoint, Message email) returns error? = external;
