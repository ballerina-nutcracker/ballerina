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

type Order record {|
    int id;
    string item;
|};

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

@ftp:ServiceConfig {path: "/in/json", fileNamePattern: ".*\\.json"}
service class JsonService {
    *ftp:Service;

    remote function onFileJson(Order content, ftp:FileInfo fileInfo) returns error? {
        recordEvent(string `json ${fileInfo.name} ${content.item}`);
    }

    remote function onError(ftp:Error err) returns error? {
        recordEvent(string `onError ${err is ftp:ContentBindingError}`);
    }
}

@ftp:ServiceConfig {path: "in/mixed"}
service class MixedService {
    *ftp:Service;

    remote function onFileText(string content, ftp:FileInfo fileInfo, ftp:Caller caller) returns error? {
        check caller->delete(fileInfo.pathDecoded);
        recordEvent(string `text ${fileInfo.name} ${content}`);
    }

    remote function onFile(byte[] content, ftp:FileInfo fileInfo) returns error? {
        recordEvent(string `bytes ${fileInfo.name} ${content.length()}`);
    }

    remote function onFileXml(xml content) returns error? {
        recordEvent(string `xml ${content.toString()}`);
    }

    remote function onFileDelete(string deletedFile) returns error? {
        recordEvent("deleted " + scrub(deletedFile));
    }
}

@ftp:ServiceConfig {path: "/in/stream"}
service class StreamService {
    *ftp:Service;

    remote function onFile(stream<byte[], error?> content, ftp:FileInfo fileInfo) returns error? {
        int total = 0;
        record {|byte[] value;|}? next = check content.next();
        while next is record {|byte[] value;|} {
            total += next.value.length();
            next = check content.next();
        }
        check content.close();
        recordEvent(string `stream ${fileInfo.name} ${total}`);
        recordEvent("info " + scrub(fileInfo.path) + " " + scrub(fileInfo.uri) + " " + fileInfo.pathDecoded + " "
            + fileInfo.depth.toString() + " " + scrub(fileInfo.friendlyURI));
    }
}

@ftp:ServiceConfig {path: "/in/panic"}
service class PanicService {
    *ftp:Service;

    remote function onFileText(string content, ftp:FileInfo fileInfo) {
        recordEvent(string `panic ${fileInfo.name}`);
        int[] empty = [];
        _ = empty[content.length()];
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

public function main() returns error? {
    ftp:Client c = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}}
    });
    check c->putText("/in/json/good.json", "{\"id\": 1, \"item\": \"pen\"}");
    check c->putText("/in/json/bad.json", "{\"id\": \"x\"}");
    check c->putText("/in/json/ignored.txt", "not json");
    check c->putText("/in/mixed/note.txt", "hello");
    check c->putBytes("/in/mixed/data.bin", [1, 2, 3, 4]);
    check c->putText("/in/mixed/doc.xml", "<a>1</a>");
    check c->putText("/in/mixed/notes.json", "{}");
    byte[] big = [];
    foreach int i in 0 ..< 10000 {
        big.push(<byte>(i % 7));
    }
    check c->putBytes("/in/stream/big.dat", big);
    check c->putText("/in/panic/boom.txt", "boom");

    ftp:Listener ftpListener = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        pollingInterval: 3600
    });
    check ftpListener.attach(new JsonService());
    check ftpListener.attach(new MixedService());
    check ftpListener.attach(new StreamService());
    check ftpListener.attach(new PanicService());

    check ftpListener.poll();
    waitFor(9);
    printEvents();
    // @output bytes data.bin 4
    // @output bytes notes.json 2
    // @output info ftp://127.0.0.1:PORT/in/stream/big.dat //127.0.0.1:PORT/in/stream/big.dat /in/stream/big.dat 4 ftp://ballerina:***@127.0.0.1:PORT/in/stream/big.dat
    // @output json good.json pen
    // @output onError true
    // @output panic boom.txt
    // @output stream big.dat 10000
    // @output text note.txt hello
    // @output xml <a>1</a>

    check c->delete("/in/mixed/data.bin");
    check ftpListener.poll();
    waitFor(2);
    printEvents();
    // @output deleted ftp://127.0.0.1:PORT/in/mixed/data.bin
    // @output deleted ftp://127.0.0.1:PORT/in/mixed/note.txt

    check ftpListener.gracefulStop();
    error? afterStop = ftpListener.poll();
    io:println(afterStop is error ? afterStop.message() : "polled"); // @output FTP listener is not initialized. Attach a service before polling.
    check c->close();
}
