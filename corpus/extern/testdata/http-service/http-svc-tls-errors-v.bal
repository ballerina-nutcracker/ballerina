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

import ballerina/http;
import ballerina/io;

function describeMessage(http:Listener|http:ListenerError result) returns string {
    if result is http:GenericListenerError {
        return "GenericListenerError: " + result.message();
    }
    return result is error ? "error: " + result.message() : "ok";
}

public function main() {
    // A directory in place of the certificate or key fails init, with jBallerina's messages.
    http:Listener|http:ListenerError certDir = new (19229, {secureSocket: {key: {certFile: "testdata/certs", keyFile: "testdata/certs/server.key"}}});
    io:println(describeMessage(certDir)); // @output GenericListenerError: File does not contain valid certificates: testdata/certs
    http:Listener|http:ListenerError keyDir = new (19229, {secureSocket: {key: {certFile: "testdata/certs/server.crt", keyFile: "testdata/certs"}}});
    io:println(describeMessage(keyDir)); // @output GenericListenerError: File does not contain valid private key: testdata/certs

    // PEM material that holds no certificate or private key fails init, as in jBallerina.
    http:Listener|http:ListenerError certNotPem = new (19229, {secureSocket: {key: {certFile: "testdata/certs/malformed.pem", keyFile: "testdata/certs/server.key"}}});
    io:println(describeMessage(certNotPem)); // @output GenericListenerError: File does not contain valid certificates: testdata/certs/malformed.pem
    http:Listener|http:ListenerError keyNotPem = new (19229, {secureSocket: {key: {certFile: "testdata/certs/server.crt", keyFile: "testdata/certs/malformed.pem"}}});
    io:println(describeMessage(keyNotPem)); // @output GenericListenerError: File does not contain valid private key: testdata/certs/malformed.pem
    http:Listener|http:ListenerError certIsKey = new (19229, {secureSocket: {key: {certFile: "testdata/certs/server.key", keyFile: "testdata/certs/server.key"}}});
    io:println(describeMessage(certIsKey)); // @output GenericListenerError: File does not contain valid certificates: testdata/certs/server.key
    http:Listener|http:ListenerError keyIsCert = new (19229, {secureSocket: {key: {certFile: "testdata/certs/server.crt", keyFile: "testdata/certs/server.crt"}}});
    io:println(describeMessage(keyIsCert)); // @output GenericListenerError: File does not contain valid private key: testdata/certs/server.crt
    http:Listener|http:ListenerError caNotPem = new (19229, {secureSocket: {key: {certFile: "testdata/certs/server.crt", keyFile: "testdata/certs/server.key"}, mutualSsl: true, cert: "testdata/certs/malformed.pem"}});
    io:println(describeMessage(caNotPem)); // @output GenericListenerError: File does not contain valid certificates: testdata/certs/malformed.pem

    // A key that does not match the certificate passes init but fails start; jBallerina starts and then fails every handshake (ballerina-platform/ballerina-library#9908).
    http:Listener|http:ListenerError mismatched = new (19230, {secureSocket: {key: {certFile: "testdata/certs/server.crt", keyFile: "testdata/certs/other.key"}}});
    io:println(describeMessage(mismatched)); // @output ok
    if mismatched is http:Listener {
        error? started = mismatched.'start();
        io:println(started is http:GenericListenerError ? "GenericListenerError: " + started.message() : started is error ? "error: " + started.message() : "ok"); // @output GenericListenerError: Listener.start: X509KeyPair: tls: private key does not match public key
    }
}
