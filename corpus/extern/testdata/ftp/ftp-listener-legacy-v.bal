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

isolated string[] events = [];

isolated function recordEvent(string event) {
    lock {
        events.push(event);
    }
}

isolated function eventCount() returns int {
    lock {
        return events.length();
    }
}

function waitFor(int count) {
    foreach int _ in 0 ..< 200 {
        if eventCount() >= count {
            return;
        }
        runtime:sleep(0.05);
    }
}

function scrub(string message) returns string {
    string port = ":" + ftpPort().toString();
    string result = message;
    int? i = result.indexOf(port);
    while i is int {
        result = result.substring(0, i) + ":PORT" + result.substring(i + port.length());
        i = result.indexOf(port);
    }
    return result;
}

function printEvents() {
    lock {
        string[] sorted = [];
        foreach string e in events {
            int i = sorted.length();
            sorted.push(e);
            while i > 0 && sorted[i - 1] > e {
                sorted[i] = sorted[i - 1];
                i -= 1;
            }
            sorted[i] = e;
        }
        foreach string e in sorted {
            io:println(e);
        }
        events.removeAll();
    }
}

service class ChangeService {
    *ftp:Service;

    remote function onFileChange(ftp:WatchEvent & readonly event, ftp:Caller caller) returns error? {
        string[] added = from ftp:FileInfo f in event.addedFiles order by f.pathDecoded select f.pathDecoded;
        recordEvent(string `change added=${added.toString()} deleted=${event.deletedFiles.length()}`);
        foreach string d in event.deletedFiles {
            recordEvent("change deleted " + scrub(d));
        }
        boolean exists = check caller->exists("/watch/a.txt");
        recordEvent("caller sees a.txt " + exists.toString());
    }
}

service class DeletedService {
    *ftp:Service;

    remote function onFileDeleted(string[] deletedFiles) returns error? {
        recordEvent(string `onFileDeleted ${deletedFiles.length()}`);
    }
}

@ftp:ServiceConfig {
    path: "/deps",
    fileDependencyConditions: [
        {targetPattern: "data_(\\d+)\\.csv", requiredFiles: ["data_$1\\.marker"]},
        {targetPattern: "any_.*", requiredFiles: ["x\\.flag", "y\\.flag"], matchingMode: ftp:ANY},
        {targetPattern: "count_.*", requiredFiles: [".*\\.flag"], matchingMode: ftp:EXACT_COUNT, requiredFileCount: 2}
    ]
}
service class DependencyService {
    *ftp:Service;

    remote function onFile(byte[] content, ftp:FileInfo fileInfo) returns error? {
        _ = content;
        recordEvent("dep " + fileInfo.name);
    }
}

@ftp:ServiceConfig {path: "/aged", fileAgeFilter: {minAge: 3600}}
service class AgeService {
    *ftp:Service;

    remote function onFile(byte[] content, ftp:FileInfo fileInfo) returns error? {
        _ = content;
        recordEvent("aged " + fileInfo.name);
    }
}

@ftp:ServiceConfig {path: "/aged"}
service class DuplicatePathService {
    *ftp:Service;

    remote function onFileText(string content) returns error? {
        _ = content;
    }
}

service class InvalidMethodService {
    *ftp:Service;

    remote function onSomething(string content) returns error? {
        _ = content;
    }
}

service class MixedHandlersService {
    *ftp:Service;

    remote function onFileChange(ftp:WatchEvent event) returns error? {
        _ = event;
    }

    remote function onFileText(string content) returns error? {
        _ = content;
    }
}

service class BothDeleteService {
    *ftp:Service;

    remote function onFileDelete(string deleted) returns error? {
        _ = deleted;
    }

    remote function onFileDeleted(string[] deleted) returns error? {
        _ = deleted;
    }
}

service class EmptyService {
    *ftp:Service;
}

service class CsvService {
    *ftp:Service;

    remote function onFileCsv(string[][] content) returns error? {
        _ = content;
    }
}

public function main() returns error? {
    ftp:Client c = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}}
    });
    check c->putText("/watch/a.txt", "a");
    check c->putText("/watch/sub/inner.txt", "inner");
    check c->putText("/watch/skip.log", "skipped by pattern");

    ftp:Listener legacy = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        path: "/watch",
        fileNamePattern: "(.*\\.txt)|sub",
        pollingInterval: 3600
    });
    check legacy.attach(new ChangeService());
    check legacy.attach(new DeletedService());

    check legacy.poll();
    waitFor(2);
    printEvents();
    // @output caller sees a.txt true
    // @output change added=["/watch/a.txt","/watch/sub/inner.txt"] deleted=0

    check legacy.poll();
    waitFor(2);
    printEvents();
    // @output caller sees a.txt true
    // @output change added=["/watch/a.txt","/watch/sub/inner.txt"] deleted=0

    check c->delete("/watch/a.txt");
    check legacy.poll();
    waitFor(4);
    printEvents();
    // @output caller sees a.txt false
    // @output change added=["/watch/sub/inner.txt"] deleted=1
    // @output change deleted ftp://127.0.0.1:PORT/watch/a.txt
    // @output onFileDeleted 1

    check c->delete("/watch/sub/inner.txt");
    check c->rmdir("/watch/sub");
    check c->delete("/watch/skip.log");
    check legacy.poll();
    waitFor(4);
    printEvents();
    // @output caller sees a.txt false
    // @output change added=[] deleted=1
    // @output change deleted ftp://127.0.0.1:PORT/watch/sub/inner.txt
    // @output onFileDeleted 1
    check legacy.gracefulStop();

    error? notInitialized = legacy.poll();
    io:println(notInitialized is error ? notInitialized.message() : "polled"); // @output FTP listener is not initialized. Attach a service before polling.

    check c->putText("/deps/data_1.csv", "1");
    check c->putText("/deps/data_1.marker", "");
    check c->putText("/deps/data_2.csv", "2");
    check c->putText("/deps/any_one", "");
    check c->putText("/deps/count_one", "");
    check c->putText("/deps/x.flag", "");
    check c->putText("/aged/new.bin", "fresh");

    ftp:Listener multi = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        pollingInterval: 3600
    });
    check multi.attach(new DependencyService());
    check multi.attach(new AgeService());
    check multi.poll();
    waitFor(4);
    printEvents();
    // @output dep any_one
    // @output dep data_1.csv
    // @output dep data_1.marker
    // @output dep x.flag

    check c->putText("/deps/y.flag", "");
    check multi.poll();
    waitFor(6);
    printEvents();
    // @output dep any_one
    // @output dep count_one
    // @output dep data_1.csv
    // @output dep data_1.marker
    // @output dep x.flag
    // @output dep y.flag

    error? duplicate = multi.attach(new DuplicatePathService());
    io:println(duplicate is ftp:InvalidConfigError ? duplicate.message() : "attached"); // @output Duplicate path '/aged' in @ftp:ServiceConfig. Each service must monitor a unique path.
    error? mixing = multi.attach(new DeletedService());
    io:println(mixing is error ? mixing.message() : "attached"); // @output All services attached to a listener must use @ftp:ServiceConfig annotation when any service uses it. The new service is missing the annotation.
    check multi.gracefulStop();

    ftp:Listener validation = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        path: "/watch"
    });
    ftp:Service[] invalid = [new InvalidMethodService(), new MixedHandlersService(), new BothDeleteService(),
        new EmptyService(), new CsvService()];
    foreach ftp:Service svc in invalid {
        error? result = validation.attach(svc);
        io:println(result is error ? result.message() : "attached");
    }
    // @output Invalid remote method. Allowed handlers: onFile, onFileText, onFileJson, onFileXml, onFileCsv (format-specific) or onFileDelete.
    // @output Cannot mix event-based handler (onFileChange) with format-specific handlers (onFile, onFileText, onFileJson, onFileXml, onFileCsv, onFileDelete).
    // @output Cannot use both 'onFileDelete' and 'onFileDeleted' methods. Use only 'onFileDelete' as 'onFileDeleted' is deprecated.
    // @output Service must define at least one handler method: onFile, onFileText, onFileJson, onFileXml, onFileCsv (format-specific) or onFileDelete.
    // @output Handling CSV content with onFileCsv is not yet supported
    error? sameType = validation.attach(new DeletedService());
    io:println(sameType is error ? sameType.message() : "attached"); // @output attached
    error? again = validation.attach(new DeletedService());
    io:println(again is error ? again.message() : "attached"); // @output Failed to register service.

    ftp:Listener missingPath = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        path: "/does/not/exist"
    });
    error? notFolder = missingPath.attach(new DeletedService());
    io:println(notFolder is error ? scrub(notFolder.message()) : "attached"); // @output FTP listener path "ftp://ballerina:***@127.0.0.1:PORT/does/not/exist" does not refer to a folder on the remote file system.

    check validation.immediateStop();
    check c->close();
}
