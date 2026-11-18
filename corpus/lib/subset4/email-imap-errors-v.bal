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

// Each failure below is detected before the client connects, so no IMAP server is needed.
public function main() returns error? {
    // jBallerina panics when secureSocket.protocol is absent (ballerina-library#9731).
    email:ImapClient|email:Error missingCert = new ("127.0.0.1", "bob", "secret", secureSocket = {cert: "/no/such/cert.pem"});
    io:println(missingCert is email:Error ? missingCert.message() : "connected"); // @output Error while initializing IMAP client properties: /no/such/cert.pem (No such file or directory)

    check io:fileWriteString("/tmp/bal_email_imap_garbage.pem", "not a certificate");
    email:ImapClient|email:Error garbageCert = new ("127.0.0.1", "bob", "secret", secureSocket = {cert: "/tmp/bal_email_imap_garbage.pem"});
    io:println(garbageCert is email:Error ? garbageCert.message() : "connected"); // @output Error while initializing IMAP client properties: Could not parse certificate: /tmp/bal_email_imap_garbage.pem
}
