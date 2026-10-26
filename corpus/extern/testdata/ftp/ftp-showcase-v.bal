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

@ftp:ServiceConfig {path: "/data/in", fileNamePattern: ".*\\.json"}
service class JsonService {
    *ftp:Service;

    remote function onFileJson(Order content, ftp:FileInfo fileInfo, ftp:Caller caller) returns error? {
        recordEvent(string `${fileInfo.name}:${content.item}`);
        check caller->delete(fileInfo.pathDecoded);
    }
}

public function main() returns error? {
    ftp:Client ftpClient = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}}
    });

    check ftpClient->mkdir("/data/in");
    check ftpClient->putText("/data/hello.txt", "Hello");
    check ftpClient->putText("/data/hello.txt", ", World!", ftp:APPEND);
    var r1 = check ftpClient->getText("/data/hello.txt");
    io:println(r1); // @output Hello, World!

    json orderJson = {id: 1, item: "pen"};
    check ftpClient->putJson("/data/order.json", orderJson);
    Order 'order = check ftpClient->getJson("/data/order.json");
    io:println('order.item); // @output pen

    check ftpClient->putXml("/data/note.xml", xml `<note>hi</note>`);
    xml note = check ftpClient->getXml("/data/note.xml");
    io:println(note); // @output <note>hi</note>

    check ftpClient->putCsv("/data/rows.csv", [["id", "name"], ["1", "a,b"]]);
    var r2 = check ftpClient->getText("/data/rows.csv");
    io:println(r2); // @output id,name
    // @output 1,"a,b"
    // @output

    check ftpClient->putBytes("/data/raw.bin", [1, 2, 3]);
    stream<byte[], error?> byteStream = check ftpClient->getBytesAsStream("/data/raw.bin");
    record {|byte[] value;|}? chunk = check byteStream.next();
    io:println(chunk?.value); // @output [1,2,3]
    check byteStream.close();

    var r3 = check ftpClient->size("/data/raw.bin");

    io:println(r3); // @output 3
    var r4 = check ftpClient->exists("/data/missing.txt");
    io:println(r4); // @output false
    var r5 = check ftpClient->isDirectory("/data/in");
    io:println(r5); // @output true
    check ftpClient->rename("/data/raw.bin", "/data/archive/raw.bin");
    check ftpClient->copy("/data/hello.txt", "/data/archive/hello.txt");
    ftp:FileInfo[] archived = check ftpClient->list("/data/archive");
    io:println(archived.length()); // @output 2

    byte[]|ftp:Error missing = ftpClient->getBytes("/data/missing.txt");
    io:println(missing is ftp:FileNotFoundError); // @output true
    ftp:Error? exists = ftpClient->mkdir("/data/in");
    io:println(exists is ftp:FileAlreadyExistsError); // @output true

    ftp:Client sftpClient = check new ({
        protocol: ftp:SFTP,
        host: "127.0.0.1",
        port: sftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}}
    });
    check sftpClient->putText("/sftp.txt", "over ssh");
    var r6 = check sftpClient->getText("/sftp.txt");
    io:println(r6); // @output over ssh
    check sftpClient->close();

    ftp:Listener ftpListener = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        pollingInterval: 0.5
    });
    ftp:Service svc = new JsonService();
    check ftpListener.attach(svc);
    check ftpListener.'start();
    Order secondOrder = {id: 2, item: "book"};
    check ftpClient->putJson("/data/in/o2.json", secondOrder);

    foreach int _ in 0 ..< 100 {
        boolean pending = check ftpClient->exists("/data/in/o2.json");
        if !pending {
            break;
        }
        runtime:sleep(0.1);
    }
    check ftpListener.gracefulStop();
    lock {
        io:println(events); // @output ["o2.json:book"]
    }
    check ftpClient->close();
}
