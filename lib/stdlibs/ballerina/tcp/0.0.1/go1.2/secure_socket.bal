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

import ballerina/crypto;

// Secure socket configuration for the TCP client.
//
// Not supported: crypto:TrustStore (PKCS12) as `cert` — a PEM certificate file path string is
// fully supported instead; supplying a crypto:TrustStore value returns an Error.
//
// Fields:
//   enable           - Enable SSL validation.
//   cert             - A PEM certificate file path that the client trusts (crypto:TrustStore not supported).
//   protocol         - SSL/TLS protocol related options.
//   ciphers          - List of ciphers to be used.
//   handshakeTimeout - SSL handshake timeout, in seconds.
//   sessionTimeout   - SSL session timeout, in seconds.
public type ClientSecureSocket record {|
    boolean enable = true;
    crypto:TrustStore|string cert?;
    record {|
        Protocol name;
        string[] versions = [];
    |} protocol?;
    string[] ciphers?;
    decimal handshakeTimeout?;
    decimal sessionTimeout?;
|};

// Secure socket configuration for the TCP listener.
//
// Not supported: crypto:KeyStore (PKCS12) as `key` — a CertKey (certificate + private key file
// pair) is fully supported instead; supplying a crypto:KeyStore value returns an Error.
//
// Fields:
//   key              - The server certificate and private key (crypto:KeyStore not supported).
//   protocol         - SSL/TLS protocol related options.
//   ciphers          - List of ciphers to be used.
//   handshakeTimeout - SSL handshake timeout, in seconds.
//   sessionTimeout   - SSL session timeout, in seconds.
public type ListenerSecureSocket record {|
    crypto:KeyStore|CertKey key;
    record {|
        Protocol name;
        string[] versions = [];
    |} protocol?;
    string[] ciphers = [];
    decimal handshakeTimeout?;
    decimal sessionTimeout?;
|};

// Represents a combination of a certificate and its private key.
//
// Fields:
//   certFile    - A file containing the certificate.
//   keyFile     - A file containing the private key in PKCS8 format.
//   keyPassword - Password of the private key if it is encrypted.
public type CertKey record {|
    string certFile;
    string keyFile;
    string keyPassword?;
|};

// Represents protocol options.
public enum Protocol {
    SSL,
    TLS
}
