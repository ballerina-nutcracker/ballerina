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

# Represents an IMAP Client, which interacts with an IMAP Server.
public isolated client class ImapClient {

    # Gets invoked during the `email:ImapClient` initialization.
    #
    # + host - Host of the IMAP Client
    # + username - Username of the IMAP Client
    # + password - Password of the IMAP Client
    # + clientConfig - Configurations for the IMAP Client
    # + return - An `email:Error` if failed while creating the client or else `()`
    public isolated function init(string host, string username, string password,
            *ImapConfiguration clientConfig) returns Error? {
        return toEmailError(initImapClientEndpoint(self, host, username, password, clientConfig));
    }

    # Reads a message.
    # ```ballerina
    # email:Message|email:Error? emailResponse = imapClient->receiveMessage();
    # ```
    #
    # + folder - Folder to read emails. The default value is `INBOX`
    # + timeout - Polling timeout period in seconds.
    # + return - An `email:Message` if reading the message is successful, `()` if there are no emails in the specified
    #            folder, or else an `email:Error` if the recipient failed to receive the message
    remote isolated function receiveMessage(string folder = DEFAULT_FOLDER, decimal timeout = 30)
            returns Message|Error? {
        Message|error? message = imapRead(self, folder, timeout);
        return message is error ? toEmailError(message) : message;
    }

    # Close the client.
    # ```ballerina
    # email:Error? closeResponse = imapClient->close();
    # ```
    #
    # + return - An `email:Error` if the recipient failed to close the client or else `()`
    remote isolated function close() returns Error? {
        return toEmailError(imapClose(self));
    }

}

isolated function initImapClientEndpoint(ImapClient clientEndpoint, string host, string username, string password,
        ImapConfiguration config) returns error? = external;

isolated function imapRead(ImapClient clientEndpoint, string folder, decimal timeout) returns Message|error? = external;

isolated function imapClose(ImapClient clientEndpoint) returns error? = external;

# Configuration of the IMAP Endpoint.
#
# + port - Port number of the IMAP server
# + security - Type of security channel
# + secureSocket - Secure socket configuration
public type ImapConfiguration record {|
    int port = 993;
    Security security = SSL;
    SecureSocket secureSocket?;
|};
