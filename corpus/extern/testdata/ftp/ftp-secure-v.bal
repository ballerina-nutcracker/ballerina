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

import ballerina/ftp;
import ballerina/io;

function sftpPort() returns int = external;

function ftpsPort() returns int = external;

function ftpsImplicitPort() returns int = external;

function certPath() returns string = external;

function keyPath() returns string = external;

function encryptedKeyPath() returns string = external;

function knownHostsPath() returns string = external;

function badKnownHostsPath() returns string = external;

function missingKeyPath() returns string = external;

function sftp(ftp:AuthConfiguration auth, string? knownHosts = (), boolean userDirIsRoot = false)
        returns ftp:Client|ftp:Error {
    ftp:ClientConfiguration config = {
        protocol: ftp:SFTP,
        host: "127.0.0.1",
        port: sftpPort(),
        auth,
        userDirIsRoot
    };
    if knownHosts is string {
        config.sftpSshKnownHosts = knownHosts;
    }
    return new (config);
}

function outcome(ftp:Client|ftp:Error c) returns string {
    if c is ftp:Error {
        string kind = c is ftp:ConnectionError ? "ConnectionError" : "Error";
        return kind;
    }
    return "connected";
}

function roundTrip(ftp:Client c, string path) returns string|error {
    check c->putText(path, "secret data");
    string text = check c->getText(path);
    check c->close();
    return text;
}

public function main() returns error? {
    ftp:Credentials creds = {username: "ballerina", password: "ballerina"};

    ftp:Client keyClient = check sftp({credentials: {username: "ballerina"}, privateKey: {path: keyPath()},
        preferredMethods: [ftp:PUBLICKEY]});
    io:println(check roundTrip(keyClient, "/key.txt")); // @output secret data

    ftp:Client encrypted = check sftp({credentials: {username: "ballerina"},
        privateKey: {path: encryptedKeyPath(), password: "keypass"}});
    io:println(check roundTrip(encrypted, "/enc.txt")); // @output secret data

    ftp:Client|ftp:Error wrongPassphrase = sftp({credentials: {username: "ballerina"},
        privateKey: {path: encryptedKeyPath(), password: "nope"}});
    io:println(outcome(wrongPassphrase)); // @output ConnectionError
    if wrongPassphrase is ftp:Error {
        io:println(wrongPassphrase.message().includes("SFTP key authentication failed for private key")); // @output true
    }

    ftp:Client|ftp:Error missingKey = sftp({credentials: {username: "ballerina"}, privateKey: {path: missingKeyPath()}});
    if missingKey is ftp:Error {
        io:println(missingKey.message().indexOf("SFTP private key file does not exist: ") == 0); // @output true
    }

    ftp:Client|ftp:Error badPassword = sftp({credentials: {username: "ballerina", password: "wrong"}});
    io:println(outcome(badPassword)); // @output ConnectionError

    ftp:Client|ftp:Error interactive = sftp({credentials: creds, preferredMethods: [ftp:KEYBOARD_INTERACTIVE, ftp:PASSWORD]});
    io:println(outcome(interactive)); // @output connected

    ftp:Client|ftp:Error noMethod = sftp({credentials: creds, preferredMethods: [ftp:GSSAPI_WITH_MIC]});
    io:println(outcome(noMethod)); // @output ConnectionError

    ftp:Client|ftp:Error emptyMethods = sftp({credentials: creds, preferredMethods: []});
    if emptyMethods is ftp:Error {
        io:println(emptyMethods.message()); // @output No preferred auth method is specified in the SFTP client configurations
    }

    ftp:Client|ftp:Error trusted = sftp({credentials: creds}, knownHostsPath());
    io:println(outcome(trusted)); // @output connected
    ftp:Client|ftp:Error untrusted = sftp({credentials: creds}, badKnownHostsPath());
    io:println(outcome(untrusted)); // @output ConnectionError
    ftp:Client|ftp:Error noFile = sftp({credentials: creds}, missingKeyPath());
    io:println(outcome(noFile)); // @output connected

    ftp:Client sftpClient = check sftp({credentials: creds}, userDirIsRoot = true);
    check sftpClient->mkdir("/tree/sub");
    check sftpClient->putText("/tree/sub/a.txt", "one");
    check sftpClient->putText("/tree/sub/a.txt", "+two", ftp:APPEND);
    string appended = check sftpClient->getText("/tree/sub/a.txt");
    io:println(appended); // @output one+two
    check sftpClient->copy("/tree/sub/a.txt", "/tree/b.txt");
    check sftpClient->rename("/tree/b.txt", "/tree/c.txt");
    ftp:FileInfo[] listed = check sftpClient->list("/tree");
    string[] names = from ftp:FileInfo f in listed order by f.name select f.name;
    io:println(names); // @output ["c.txt","sub"]
    foreach ftp:FileInfo info in listed {
        if info.name == "c.txt" {
            io:println(info.scheme, " ", info.size, " ", info.isReadable); // @output sftp 7 true
        }
    }
    stream<byte[], error?> chunks = check sftpClient->getBytesAsStream("/tree/c.txt");
    record {|byte[] value;|}? first = check chunks.next();
    io:println(first?.value); // @output [111,110,101,43,116,119,111]
    check chunks.close();
    check sftpClient->delete("/tree/c.txt");
    check sftpClient->rmdir("/tree");
    boolean treeExists = check sftpClient->exists("/tree");
    io:println(treeExists); // @output false
    check sftpClient->close();

    ftp:Client explicit = check new ({
        protocol: ftp:FTPS,
        host: "127.0.0.1",
        port: ftpsPort(),
        auth: {credentials: creds, secureSocket: {cert: certPath()}}
    });
    io:println(check roundTrip(explicit, "/explicit.txt")); // @output secret data

    ftp:Client implicit = check new ({
        protocol: ftp:FTPS,
        host: "127.0.0.1",
        port: ftpsImplicitPort(),
        auth: {credentials: creds, secureSocket: {cert: certPath(), mode: ftp:IMPLICIT}}
    });
    io:println(check roundTrip(implicit, "/implicit.txt")); // @output secret data

    ftp:Client|ftp:Error untrustedTls = new ({
        protocol: ftp:FTPS,
        host: "127.0.0.1",
        port: ftpsPort(),
        auth: {credentials: creds}
    });
    io:println(outcome(untrustedTls)); // @output ConnectionError

    ftp:Client|ftp:Error keyStoreTls = new ({
        protocol: ftp:FTPS,
        host: "127.0.0.1",
        port: ftpsPort(),
        auth: {
            credentials: creds,
            secureSocket: {
                key: {path: "testdata/crypto-keystore.p12", password: "secret"},
                cert: {path: "testdata/crypto-keystore.p12", password: "secret"}
            }
        }
    });
    io:println(outcome(keyStoreTls)); // @output ConnectionError

    ftp:Client|ftp:Error badKeyStore = new ({
        protocol: ftp:FTPS,
        host: "127.0.0.1",
        port: ftpsPort(),
        auth: {credentials: creds, secureSocket: {key: {path: "testdata/crypto-keystore.p12", password: "wrong"}}}
    });
    if badKeyStore is ftp:Error {
        io:println(badKeyStore.message().indexOf("Failed to configure SSL/TLS certificates for FTPS: ") == 0); // @output true
    }
}
