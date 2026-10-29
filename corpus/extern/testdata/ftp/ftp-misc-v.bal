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
import ballerina/lang.runtime;

function ftpPort() returns int = external;

function sftpPort() returns int = external;

function hashedKnownHostsPath() returns string = external;

function ftpConnections() returns int = external;

@ftp:ServiceConfig {path: "/missing"}
service class MissingPathService {
    *ftp:Service;

    remote function onFileChange(ftp:WatchEvent event) returns error? {
        io:println(event.addedFiles.length());
    }
}

type Profile record {|
    int id;
    string? nickname;
    string email?;
|};

isolated int changes = 0;

service class CountingService {
    *ftp:Service;

    remote function onFileChange(ftp:WatchEvent event) returns error? {
        if event.addedFiles.length() > 0 {
            lock {
                changes += 1;
            }
        }
    }
}

isolated int inboxChanges = 0;
isolated int outboxChanges = 0;

@ftp:ServiceConfig {path: "/inbox"}
service class InboxService {
    *ftp:Service;

    remote function onFileChange(ftp:WatchEvent event) returns error? {
        if event.addedFiles.length() > 0 {
            lock {
                inboxChanges += 1;
            }
        }
    }
}

@ftp:ServiceConfig {path: "/outbox"}
service class OutboxService {
    *ftp:Service;

    remote function onFileChange(ftp:WatchEvent event) returns error? {
        if event.addedFiles.length() > 0 {
            lock {
                outboxChanges += 1;
            }
        }
    }
}

function waitForOutbox(int target) {
    foreach int _ in 0 ..< 200 {
        int seen;
        lock {
            seen = outboxChanges;
        }
        if seen >= target {
            return;
        }
        runtime:sleep(0.05);
    }
}

function inboxCount() returns int {
    lock {
        return inboxChanges;
    }
}

function outboxCount() returns int {
    lock {
        return outboxChanges;
    }
}

public function main() returns error? {
    ftp:Client lax = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        laxDataBinding: true,
        socketConfig: {ftpDataTimeout: 30, ftpSocketTimeout: 30},
        connectTimeout: 10
    });
    check lax->putText("/home/profile.json", "{\"id\": 3, \"email\": null, \"extra\": true, \"list\": [1]}");
    Profile profile = check lax->getJson("/home/profile.json");
    io:println(profile); // @output {"id":3,"nickname":null}
    check lax->putText("/home/profiles.json", "[{\"id\": 1}, {\"id\": 2, \"nickname\": \"b\"}]");
    Profile[] profiles = check lax->getJson("/home/profiles.json");
    io:println(profiles); // @output [{"id":1,"nickname":null},{"id":2,"nickname":"b"}]

    foreach int _ in 0 ..< 5 {
        stream<byte[] & readonly, io:Error?> chunks = check lax->get("/home/profiles.json");
        record {|byte[] & readonly value;|}? chunk = check chunks.next();
        while chunk !is () {
            chunk = check chunks.next();
        }
    }
    _ = check lax->get("/home/profile.json");
    _ = check lax->get("/home/profiles.json");
    io:println(ftpConnections() <= 2); // @output true

    ftp:Client strict = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        userDirIsRoot: true,
        fileTransferMode: ftp:ASCII
    });
    Profile|ftp:Error strictProfile = strict->getJson("/home/profile.json");
    io:println(strictProfile is ftp:ContentBindingError); // @output true
    string text = check strict->getText("/home/profile.json");
    io:println(text.length() > 0); // @output true
    check strict->close();

    ftp:Client hashed = check new ({
        protocol: ftp:SFTP,
        host: "127.0.0.1",
        port: sftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        sftpSshKnownHosts: hashedKnownHostsPath()
    });
    boolean root = check hashed->isDirectory("/");
    io:println(root); // @output true
    check hashed->close();

    ftp:Client|ftp:Error refused = new ({host: "127.0.0.1", port: 1});
    if refused is ftp:Error {
        io:println(refused is ftp:ConnectionError); // @output true
        io:println(refused.message().indexOf("Error while connecting to the FTP server with URL: ftp://anonymous:***@127.0.0.1:1/. ") == 0); // @output true
    }

    ftp:Listener legacyDown = check new ({host: "127.0.0.1", port: 1, path: "/"});
    error? legacyAttach = legacyDown.attach(new CountingService());
    if legacyAttach is error {
        io:println(legacyAttach.message().indexOf("Failed to initialize File server connector. ") == 0); // @output true
    }

    ftp:Listener watcher = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        path: "/home",
        pollingInterval: 0.2
    });
    check watcher.attach(new CountingService());
    check watcher.'start();
    foreach int _ in 0 ..< 100 {
        int seen;
        lock {
            seen = changes;
        }
        if seen >= 2 {
            break;
        }
        runtime:sleep(0.05);
    }
    check watcher.detach(new CountingService());
    lock {
        io:println(changes >= 2); // @output true
    }
    runtime:sleep(0.5);
    check watcher.gracefulStop();

    check lax->putText("/inbox/a.txt", "a");
    check lax->putText("/outbox/b.txt", "b");
    ftp:Listener paths = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        pollingInterval: 0.2
    });
    check paths.attach(new InboxService(), ["inbox", "orders"]);
    check paths.attach(new OutboxService());
    check paths.'start();
    waitForOutbox(1);
    io:println(inboxCount() >= 1); // @output true
    check paths.detach(new InboxService());
    waitForOutbox(outboxCount() + 2);
    int inboxAfterDetach = inboxCount();
    int outboxAfterDetach = outboxCount();
    waitForOutbox(outboxAfterDetach + 2);
    io:println(outboxCount() >= outboxAfterDetach + 2); // @output true
    io:println(inboxCount() == inboxAfterDetach); // @output true
    check paths.detach(new InboxService());
    check paths.gracefulStop();

    ftp:Listener retried = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        path: "/home"
    });
    error? missingPath = retried.attach(new MissingPathService());
    io:println(missingPath is error); // @output true
    error? legacyAfterFailure = retried.attach(new CountingService());
    io:println(legacyAfterFailure is ()); // @output true
    check retried.detach(new CountingService());
    check lax->close();
}
