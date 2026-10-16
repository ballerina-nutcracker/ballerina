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

import ballerina/io;
import ballerina/ldap;

function connect(ldap:ClientSecureSocket secureSocket) returns ldap:Client|ldap:Error {
    return new ({hostName: "127.0.0.1", port: 1, domainName: "cn=admin", password: "pw", clientSecureSocket: secureSocket});
}

public function main() {
    ldap:Client|ldap:Error trustStore = connect({cert: {path: "ts.p12", password: "pw"}});
    if trustStore is ldap:Error {
        io:println(trustStore.message()); // @output clientSecureSocket.cert: crypto:TrustStore is not yet supported; use a PEM certificate file path string instead
    }

    ldap:Client|ldap:Error missingCert = connect({cert: "no-such-cert.pem"});
    io:println(missingCert is ldap:Error); // @output true

    ldap:Client|ldap:Error badVersion = connect({tlsVersions: ["TLSv9"]});
    if badVersion is ldap:Error {
        io:println(badVersion.message()); // @output clientSecureSocket.tlsVersions: unsupported TLS version 'TLSv9'
    }
}
