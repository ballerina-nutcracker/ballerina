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

// Each failure below is detected before the listener connects, so no mail server is needed.
// jBallerina panics instead of returning these errors (ballerina-library#9776).
public function main() {
    email:PopListener|email:Error pop = new ({host: "127.0.0.1", username: "bob", password: "secret",
        secureSocket: {cert: "/no/such/cert.pem"}});
    io:println(pop is email:Error ? pop.message() : "connected"); // @output Error while initializing POP3 client properties: /no/such/cert.pem (No such file or directory)

    email:ImapListener|email:Error imap = new ({host: "127.0.0.1", username: "bob", password: "secret",
        secureSocket: {cert: "/no/such/cert.pem"}});
    io:println(imap is email:Error ? imap.message() : "connected"); // @output Error while initializing IMAP client properties: /no/such/cert.pem (No such file or directory)
}
