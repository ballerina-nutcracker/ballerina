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

import ballerina/ftp;
import ballerina/io;
import ballerina/lang.runtime;

function ftpPort() returns int = external;

function sftpPort() returns int = external;

function hashedKnownHostsPath() returns string = external;

function ftpConnections() returns int = external;

function awaitStderr(string text, int count) returns error? = external;

// TODO(#867): move the annotated services to service object constructors; @ftp:ServiceConfig on a
// service class compiles only while #1445 is open.
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

service class CallerService {
    *ftp:Service;

    remote function onFileChange(ftp:WatchEvent event, ftp:Caller caller) returns error? {
        _ = event;
        _ = caller;
    }
}

isolated string[] inboxSeen = [];
isolated string[] outboxSeen = [];

@ftp:ServiceConfig {path: "/inbox"}
service class InboxService {
    *ftp:Service;

    remote function onFileChange(ftp:WatchEvent event) returns error? {
        foreach ftp:FileInfo f in event.addedFiles {
            lock {
                inboxSeen.push(f.name);
            }
        }
    }
}

@ftp:ServiceConfig {path: "/outbox"}
service class OutboxService {
    *ftp:Service;

    remote function onFileChange(ftp:WatchEvent event) returns error? {
        foreach ftp:FileInfo f in event.addedFiles {
            lock {
                outboxSeen.push(f.name);
            }
        }
    }
}

function seenCount(boolean outbox, string name) returns int {
    int count = 0;
    if outbox {
        lock {
            foreach string seen in outboxSeen {
                count += seen == name ? 1 : 0;
            }
        }
    } else {
        lock {
            foreach string seen in inboxSeen {
                count += seen == name ? 1 : 0;
            }
        }
    }
    return count;
}

function waitForSeen(boolean outbox, string name, int target) {
    foreach int _ in 0 ..< 200 {
        if seenCount(outbox, name) >= target {
            return;
        }
        runtime:sleep(0.05);
    }
}

function waitForChanges(int target) {
    foreach int _ in 0 ..< 200 {
        int seen;
        lock {
            seen = changes;
        }
        if seen >= target {
            return;
        }
        runtime:sleep(0.05);
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

    // A later get closes the earlier stream, which then fails instead of ending early
    // (ballerina-platform/ballerina-library#9830).
    byte[] big = [];
    foreach int i in 0 ..< 8193 {
        big.push(<byte>(i % 7));
    }
    check lax->putBytes("/home/big.bin", big);
    stream<byte[] & readonly, io:Error?> first = check lax->get("/home/big.bin");
    stream<byte[] & readonly, io:Error?> second = check lax->get("/home/profile.json");
    record {|byte[] & readonly value;|}|io:Error? firstChunk = first.next();
    io:println(firstChunk is record {|byte[] & readonly value;|} ? firstChunk.value.length() : -1); // @output 8192
    record {|byte[] & readonly value;|}|io:Error? superseded = first.next();
    io:println(superseded is error ? superseded.message() : "ended"); // @output Error occurred while reading the file: the stream was closed by a later get on the same client
    io:println(first.close() is ()); // @output true
    record {|byte[] & readonly value;|}? secondChunk = check second.next();
    io:println(secondChunk is () ? -1 : secondChunk.value.length() > 0); // @output true
    check second.close();
    io:println(ftpConnections() <= 2); // @output true

    // An anonymous login sends the password "anonymous", as commons-vfs does.
    ftp:Client anonymous = check new ({host: "127.0.0.1", port: ftpPort()});
    boolean anonymousSees = check anonymous->exists("/home/profile.json");
    io:println(anonymousSees); // @output true
    check anonymous->close();

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

    // The password is masked also for an empty username (ballerina-platform/ballerina-library#9829).
    ftp:Client|ftp:Error emptyUser = new ({host: "127.0.0.1", port: 1, auth: {credentials: {username: "", password: "s3cret"}}});
    if emptyUser is ftp:Error {
        io:println(emptyUser.message().indexOf("Error while connecting to the FTP server with URL: ftp://:***@127.0.0.1:1/. ") == 0,
            " ", emptyUser.message().includes("s3cret")); // @output true false
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
    CountingService counting = new;
    check watcher.attach(counting);
    check watcher.'start();
    waitForChanges(1);
    check lax->putText("/home/second.txt", "2");
    waitForChanges(2);
    check watcher.detach(counting);
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
    InboxService inbox = new;
    check paths.attach(inbox, ["inbox", "orders"]);
    check paths.attach(new OutboxService());
    check paths.'start();
    waitForSeen(false, "a.txt", 1);
    waitForSeen(true, "b.txt", 1);
    io:println(seenCount(false, "a.txt") >= 1, " ", seenCount(true, "b.txt") >= 1); // @output true true
    // A poll lists the inbox before the outbox, and polls do not overlap, so the second poll that
    // reports e.txt has listed the inbox after d.txt was written.
    check paths.detach(inbox);
    check lax->putText("/inbox/d.txt", "d");
    check lax->putText("/outbox/e.txt", "e");
    waitForSeen(true, "e.txt", 2);
    io:println(seenCount(true, "e.txt") >= 2, " ", seenCount(false, "d.txt")); // @output true 0
    check paths.detach(inbox);
    check paths.gracefulStop();

    // The listener keeps the configuration it was created with, not later changes to the record.
    ftp:ListenerConfiguration watchConfig = {
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        path: "/home",
        pollingInterval: 3600
    };
    ftp:Listener snapshot = check new (watchConfig);
    watchConfig.port = 1;
    watchConfig.fileDependencyConditions = [{targetPattern: "[", requiredFiles: []}];
    error? snapshotAttach = snapshot.attach(new CallerService());
    io:println(snapshotAttach is error ? snapshotAttach.message() : "attached"); // @output attached
    check snapshot.gracefulStop();

    // A started listener without a service logs each scheduled poll, as jBallerina does.
    ftp:Listener idle = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        pollingInterval: 3600
    });
    check idle.'start();
    check awaitStderr("FTP listener is not initialized. Attach a service before polling.", 1);
    check idle.gracefulStop();

    ftp:Listener retried = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        path: "/home"
    });
    error? missingPath = retried.attach(new MissingPathService());
    io:println(missingPath is error); // @output true
    CountingService retriedCounting = new;
    error? legacyAfterFailure = retried.attach(retriedCounting);
    io:println(legacyAfterFailure is ()); // @output true
    check retried.detach(retriedCounting);
    check lax->close();
}
