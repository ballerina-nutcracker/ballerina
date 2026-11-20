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

import ballerina/email;
import ballerina/io;
import ballerina/mime;

// Every failure below is detected before the client connects, so no SMTP server is needed.
function report(string label, email:Error? result) {
    io:println(label, ": [", result is email:Error ? result.message() : "sent", "]");
}

public function main() returns error? {
    email:SmtpClient|email:Error usernameOnly = new ("127.0.0.1", "alice");
    report("username-only", usernameOnly is email:Error ? usernameOnly : ()); // @output username-only: [Invalid configuration: provide 'username'+'password' for basic auth, 'username'+'auth' for OAuth2, or neither for unauthenticated mode.]
    email:SmtpClient|email:Error passwordOnly = new ("127.0.0.1", password = "secret");
    report("password-only", passwordOnly is email:Error ? passwordOnly : ()); // @output password-only: [Invalid configuration: provide 'username'+'password' for basic auth, 'username'+'auth' for OAuth2, or neither for unauthenticated mode.]

    check io:fileWriteString("/tmp/bal_email_key.pem",
        "-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n");
    // jBallerina panics here: without secureSocket.protocol (ballerina-library#9731), and on an unreadable cert (ballerina-library#9732).
    email:SmtpClient|email:Error keyAsCert = new ("127.0.0.1", "alice", "secret",
        secureSocket = {cert: "/tmp/bal_email_key.pem"});
    report("pem-without-certificate", keyAsCert is email:Error ? keyAsCert : ()); // @output pem-without-certificate: [Could not parse certificate: /tmp/bal_email_key.pem]

    email:SmtpClient anonymous = check new ("127.0.0.1", port = 2525, security = email:START_TLS_NEVER);
    email:SmtpClient basic = check new ("127.0.0.1", "alice", "secret", port = 2525, security = email:START_TLS_NEVER);

    email:Error? notText = basic->sendMessage({to: "bob@localhost", subject: "s", contentType: "application/json"});
    report("not-text", notText); // @output not-text: [Content type of the email should be text.]
    // jBallerina's send skips the check that sendMessage makes (ballerina-library#9752).
    email:Error? sendNotText = basic->send("bob@localhost", "s", "alice@localhost", "{}", contentType = "application/json");
    report("send-not-text", sendNotText); // @output send-not-text: [Content type of the email should be text.]
    email:Error? invalidType = basic->send("bob@localhost", "s", "alice@localhost", "b", contentType = "text");
    report("invalid-content-type", invalidType); // @output invalid-content-type: [Invalid content type: text]

    email:Error? missingFrom = anonymous->sendMessage({to: "bob@localhost", subject: "s", body: "b"});
    report("missing-from", missingFrom); // @output missing-from: [The 'from' field is mandatory when using SMTP client without authentication. Please specify the 'from' address in the email message.]

    email:Error? whitespace = basic->send("not an address", "s", "alice@localhost", "b");
    report("whitespace-address", whitespace); // @output whitespace-address: [Local address contains control or whitespace]
    email:Error? list = basic->send("a@localhost, b@localhost", "s", "alice@localhost", "b");
    report("address-list", list); // @output address-list: [Illegal address]
    email:Error? badCc = basic->send("bob@localhost", "s", "alice@localhost", "b", cc = "<broken");
    report("bad-cc", badCc); // @output bad-cc: [Illegal address]
    email:Error? badBcc = basic->send("bob@localhost", "s", "alice@localhost", "b", bcc = [""]);
    report("bad-bcc", badBcc); // @output bad-bcc: [Illegal address]
    email:Error? badReplyTo = basic->send("bob@localhost", "s", "alice@localhost", "b", replyTo = "a b");
    report("bad-reply-to", badReplyTo); // @output bad-reply-to: [Local address contains control or whitespace]
    email:Error? badSender = basic->send("bob@localhost", "s", "alice@localhost", "b", sender = "x y");
    report("bad-sender", badSender); // @output bad-sender: [Local address contains control or whitespace]
    email:Error? badFrom = basic->send("bob@localhost", "s", "@", "b");
    report("bad-from", badFrom); // @output bad-from: [Illegal address]

    // jBallerina sends a header name that is not a field name as a malformed line (ballerina-library#9771).
    email:Error? badHeader = basic->send("bob@localhost", "s", "alice@localhost", "b", headers = {"Bad Name": "v"});
    report("bad-header-name", badHeader); // @output bad-header-name: [Invalid header name: Bad Name]

    email:Error? clearedTo = basic->send("bob@localhost", "s", "alice@localhost", "b", headers = {"To": ""});
    report("cleared-to", clearedTo); // @output cleared-to: [Error while sending the message to SMTP server : No recipient addresses ]

    email:Error? noRecipients = basic->sendMessage({to: [], subject: "s", body: "b"});
    report("no-recipients", noRecipients); // @output no-recipients: [Error while sending the message to SMTP server : No recipient addresses ]

    email:Error? badCharset = basic->sendMessage({to: "bob@localhost", subject: "s", body: "b",
        contentType: "text/plain; charset=x-no-such-charset"});
    report("bad-charset", badCharset); // @output bad-charset: [Unsupported charset: x-no-such-charset]

    // jBallerina ignores the content type and sends an empty part for a multipart one (ballerina-library#9735).
    email:Error? multipartFile = basic->send("bob@localhost", "s", "alice@localhost", "b",
        attachments = {filePath: "/tmp/bal_email_attachment.txt", contentType: "multipart/mixed"});
    report("multipart-file", multipartFile); // @output multipart-file: [The content type of a file attachment cannot be multipart: multipart/mixed]
    email:Error? invalidAttachmentType = basic->send("bob@localhost", "s", "alice@localhost", "b",
        attachments = {filePath: "/tmp/bal_email_attachment.txt", contentType: "not a type"});
    report("invalid-attachment-type", invalidAttachmentType); // @output invalid-attachment-type: [Invalid content type for the attachment '/tmp/bal_email_attachment.txt': not a type]
    email:Error? missingFile = basic->send("bob@localhost", "s", "alice@localhost", "b",
        attachments = [{filePath: "/tmp/bal_email_no_such_file.txt", contentType: "text/plain"}]);
    report("missing-file", missingFile); // @output missing-file: [IOException while sending message]
    io:println(missingFile is email:Error && missingFile.cause() is error); // @output true

    mime:Entity injected = new;
    injected.setText("x");
    injected.setHeader("X-A: x\r\nBcc: eve@localhost\r\nX-B", "v");
    email:Error? injectedName = basic->send("bob@localhost", "s", "alice@localhost", "b", attachments = injected);
    io:println(injectedName is email:Error && injectedName.message().indexOf("Invalid header name: X-A: x") == 0); // @output true

    mime:Entity badMultipart = new;
    badMultipart.setBodyParts([new mime:Entity()], "multipart/mixed");
    badMultipart.setHeader("Content-Type", "multipart/;;");
    email:Error? invalidEntity = basic->send("bob@localhost", "s", "alice@localhost", "b", attachments = badMultipart);
    report("invalid-entity-type", invalidEntity); // @output invalid-entity-type: [Invalid content type for the attachment: multipart/;;]
}
