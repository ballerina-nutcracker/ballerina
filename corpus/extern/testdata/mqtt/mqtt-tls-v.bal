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

// Runs against the in-process broker started by corpus/extern/mqtt_test.go, which has a TLS, a
// mutual TLS and a mutual TLS with credentials listener. Paths are relative to corpus/extern.
import ballerina/crypto;
import ballerina/io;
import ballerina/mqtt;

isolated function brokerUrl(string listenerId) returns string = external;

final string TLS_URL = brokerUrl("tls");
final string MTLS_URL = brokerUrl("mtls");
final string MTLS_PASSWORD_URL = brokerUrl("mtls-password");

final string BROKER_URL = brokerUrl("anon");

const SERVER_CERT = "testdata/mqtt/certs/server.crt";
const CLIENT_CERT = "testdata/mqtt/certs/client.crt";
const CLIENT_KEY = "testdata/mqtt/certs/client.key";
const CLIENT_ENCRYPTED_KEY = "testdata/mqtt/certs/client-encrypted.key";
const CLIENT_CORRUPT_KEY = "testdata/mqtt/certs/client-corrupt.key";

function connect(string label, string url, string clientId, mqtt:ConnectionConfiguration connectionConfig) returns error? {
    mqtt:Client|mqtt:Error c = new (url, clientId, connectionConfig = connectionConfig);
    if c is mqtt:Error {
        io:println(label, ": ", c.message());
        return;
    }
    mqtt:DeliveryToken token = check c->publish("tls-v/t", {payload: "secure".toBytes()});
    io:println(label, ": published to ", token.topic);
    check c->disconnect();
    check c->close();
}

public function main() returns error? {
    check connect("tls", TLS_URL, "tls-v-1", {secureSocket: {cert: SERVER_CERT}}); // @output tls: published to tls-v/t
    // @output tls 1.3: published to tls-v/t
    check connect("tls 1.3", TLS_URL, "tls-v-2", {secureSocket: {cert: SERVER_CERT, protocol: {name: mqtt:TLS, version: "1.3"}}});
    // @output unknown protocol version: published to tls-v/t
    check connect("unknown protocol version", TLS_URL, "tls-v-3", {secureSocket: {cert: SERVER_CERT, protocol: {name: mqtt:SSL, version: "9"}}});
    // @output mtls: published to tls-v/t
    check connect("mtls", MTLS_URL, "tls-v-4", {secureSocket: {cert: SERVER_CERT, key: {certFile: CLIENT_CERT, keyFile: CLIENT_KEY}}});
    // @output mtls ignores password of a plain key: published to tls-v/t
    check connect("mtls ignores password of a plain key", MTLS_URL, "tls-v-5", {secureSocket: {cert: SERVER_CERT, key: {certFile: CLIENT_CERT, keyFile: CLIENT_KEY, keyPassword: "ballerina"}}});
    // @output mtls with credentials: published to tls-v/t
    check connect("mtls with credentials", MTLS_PASSWORD_URL, "tls-v-6", {
        username: "ballerina",
        password: "ballerinamqtt",
        secureSocket: {cert: SERVER_CERT, key: {certFile: CLIENT_CERT, keyFile: CLIENT_KEY}}
    });

    // @output untrusted server: Untranslated MqttException - RC: 0
    check connect("untrusted server", TLS_URL, "tls-v-7", {});
    // @output mtls without client key: Untranslated MqttException - RC: 0
    check connect("mtls without client key", MTLS_URL, "tls-v-8", {secureSocket: {cert: SERVER_CERT}});
    // @output tls on tcp uri: The specified SocketFactory type does not match the broker URI
    check connect("tls on tcp uri", BROKER_URL, "tls-v-9", {secureSocket: {cert: SERVER_CERT}});
    // @output missing cert: Failed to get the public key from Crypto API. Certificate file not found at: testdata/mqtt/certs/missing.crt
    check connect("missing cert", TLS_URL, "tls-v-10", {secureSocket: {cert: "testdata/mqtt/certs/missing.crt"}});
    // @output missing key: Failed to get the private key from Crypto API. Key file not found at: testdata/mqtt/certs/missing.key
    check connect("missing key", MTLS_URL, "tls-v-11", {secureSocket: {cert: SERVER_CERT, key: {certFile: CLIENT_CERT, keyFile: "testdata/mqtt/certs/missing.key"}}});
    // @output invalid cert: Failed to get the public key from Crypto API. Unable to do public key operations: x509: malformed tbs certificate
    check connect("invalid cert", TLS_URL, "tls-v-12", {secureSocket: {cert: CLIENT_KEY}});
    // @output invalid key: Failed to get the private key from Crypto API. Failed to parse private key information from the given input
    check connect("invalid key", MTLS_URL, "tls-v-13", {secureSocket: {cert: SERVER_CERT, key: {certFile: CLIENT_CERT, keyFile: "testdata/mqtt/mqtt-tls-v.bal"}}});
    // @output corrupt key: Failed to get the private key from Crypto API. Failed to parse private key information from the given input
    check connect("corrupt key", MTLS_URL, "tls-v-17", {secureSocket: {cert: SERVER_CERT, key: {certFile: CLIENT_CERT, keyFile: CLIENT_CORRUPT_KEY}}});

    // Not yet supported: encrypted private keys and PKCS#12 trust/key stores.
    // @output encrypted key: Failed to get the private key from Crypto API. Encrypted private keys are not supported: testdata/mqtt/certs/client-encrypted.key
    check connect("encrypted key", MTLS_URL, "tls-v-14", {secureSocket: {cert: SERVER_CERT, key: {certFile: CLIENT_CERT, keyFile: CLIENT_ENCRYPTED_KEY, keyPassword: "ballerina"}}});
    crypto:TrustStore trustStore = {path: "truststore.p12", password: "ballerina"};
    // @output trust store: crypto:TrustStore is not supported as the mqtt:SecureSocket cert; provide a PEM certificate file path
    check connect("trust store", TLS_URL, "tls-v-15", {secureSocket: {cert: trustStore}});
    crypto:KeyStore keyStore = {path: "keystore.p12", password: "ballerina"};
    // @output key store: crypto:KeyStore is not supported as the mqtt:SecureSocket key; provide an mqtt:CertKey
    check connect("key store", MTLS_URL, "tls-v-16", {secureSocket: {cert: SERVER_CERT, key: keyStore}});
}
