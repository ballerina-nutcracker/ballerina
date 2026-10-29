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

function ftpPort() returns int = external;

// Error messages embed the server port, which differs per run.
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

function show(string label, any|error e) {
    if e is error {
        io:println(label, ": ", scrub(e.message()));
    } else {
        io:println(label, ": ok");
    }
}

type Item record {|
    int id;
    string? note;
|};

type OpenItem record {
    int id;
};

// CSV rows end with the platform line separator, as in jBallerina; drop the carriage returns
// Windows adds so the expected output is the same everywhere.
function unixLines(string text) returns string {
    string result = text;
    int? i = result.indexOf("\r");
    while i is int {
        result = result.substring(0, i) + result.substring(i + 1);
        i = result.indexOf("\r");
    }
    return result;
}

public function main() returns error? {
    ftp:Client c = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}}
    });

    check c->putBytes("/a/b/c.bin", [0, 255, 128]);
    byte[] bytes = check c->getBytes("/a/b/c.bin");
    io:println(bytes); // @output [0,255,128]

    check c->putText("t.txt", "relative");
    string text = check c->getText("/t.txt");
    io:println(text); // @output relative

    check c->putBytes("/bad.txt", [104, 105, 255, 33]);
    string replaced = check c->getText("/bad.txt");
    io:println(replaced.length()); // @output 4

    json document = {"b": 1, "a": [1.5, true, (), "x\"y\n"], "nested": {"k": "v"}};
    check c->putJson("/doc.json", document);
    string raw = check c->getText("/doc.json");
    io:println(raw); // @output {"b":1, "a":[1.5, true, null, "x\"y\n"], "nested":{"k":"v"}}
    json parsed = check c->getJson("/doc.json");
    io:println(parsed); // @output {"b":1,"a":[1.5,true,null,"x\"y\n"],"nested":{"k":"v"}}

    Item item = {id: 7, note: ()};
    check c->putJson("/item.json", item);
    Item boundItem = check c->getJson("/item.json");
    io:println(boundItem); // @output {"id":7,"note":null}

    check c->putText("/extra.json", "{\"id\": 1, \"note\": \"n\", \"x\": 2}");
    Item|ftp:Error closed = c->getJson("/extra.json");
    io:println(closed is ftp:ContentBindingError); // @output true
    OpenItem open = check c->getJson("/extra.json");
    io:println(open); // @output {"id":1,"note":"n","x":2}

    check c->putText("/broken.json", "{\"id\": ");
    json|ftp:Error broken = c->getJson("/broken.json");
    if broken is ftp:ContentBindingError {
        io:println(broken); // @output error ContentBindingError ("invalid JSON document: unexpected end of input",filePath="/broken.json",content=[123,34,105,100,34,58,32])
    }

    check c->putText("/data.xml", "<root><a>1</a></root>");
    xml x = check c->getXml("/data.xml");
    io:println(x); // @output <root><a>1</a></root>
    check c->putText("/bad.xml", "<root>");
    xml|ftp:Error badXml = c->getXml("/bad.xml");
    io:println(badXml is ftp:ContentBindingError); // @output true

    record {}[] rows = [{"name": "a", "qty": 1, "note": "x,y"}, {"name": "b\"c", "qty": 2}];
    check c->putCsv("/rows.csv", rows);
    check c->putCsv("/rows.csv", rows, ftp:APPEND);
    string csv = check c->getText("/rows.csv");
    io:print(unixLines(csv)); // @output name,qty,note
    // @output a,1,"x,y"
    // @output "b""c",2,
    // @output a,1,"x,y"
    // @output "b""c",2,
    string[][] empty = [];
    check c->putCsv("/empty.csv", empty);
    int emptySize = check c->size("/empty.csv");
    io:println(emptySize); // @output 0
    string[][]|ftp:Error csvRead = c->getCsv("/rows.csv");
    show("getCsv", csvRead); // @output getCsv: Reading CSV content is not yet supported

    Item recordXml = {id: 1, note: ()};
    var r1 = c->putXml("/r.xml", recordXml);
    show("putXml record", r1); // @output putXml record: Failed to convert record to XML: converting a record to XML is not yet supported

    ftp:FileInfo[] files = check c->list("/a/b");
    ftp:FileInfo f = files[0];
    io:println(f.path, " ", f.name, " ", f.extension, " ", f.size, " ", f.isFile, " ", f.isFolder, " ", f.depth);
    // @output /a/b/c.bin c.bin bin 3 true false 4
    io:println(scrub(f.uri)); // @output ftp://ballerina:ballerina@127.0.0.1:PORT/a/b/c.bin
    io:println(scrub(f.friendlyURI)); // @output ftp://ballerina:***@127.0.0.1:PORT/a/b/c.bin
    io:println(scrub(f.rootURI), " ", f.scheme, " ", f.fileType, " ", f.lastModifiedTimestamp > 0);
    // @output ftp://ballerina:ballerina@127.0.0.1:PORT/ ftp file true
    check c->putText("/my dir/a b#1.txt", "s");
    ftp:FileInfo[] spaced = check c->list("/my dir");
    io:println(scrub(spaced[0].friendlyURI), " ", scrub(spaced[0].uri), " ", spaced[0].pathDecoded);
    // @output ftp://ballerina:***@127.0.0.1:PORT/my%20dir/a%20b%231.txt ftp://ballerina:ballerina@127.0.0.1:PORT/my%20dir/a%20b%231.txt /my dir/a b#1.txt
    var spacedMissing = c->getText("/my dir/no such.txt");
    show("read spaced missing", spacedMissing);
    // @output read spaced missing: Failed to read file: ftp://ballerina:***@127.0.0.1:PORT/my%20dir/no%20such.txt not found
    ftp:FileInfo[] dirs = check c->list("/a");
    io:println(dirs[0].isFolder, " ", dirs[0].size, " ", dirs[0].fileType); // @output true 0 folder

    boolean isDir = check c->isDirectory("/a/b/c.bin");
    io:println(isDir); // @output false
    var r2 = c->isDirectory("/nope");
    show("isDirectory missing", r2); // @output isDirectory missing: /nope does not exist to check if it is a directory.
    var r3 = c->size("/a");
    show("size dir", r3); // @output size dir: Could not determine the size of "ftp://ballerina:***@127.0.0.1:PORT/a" because it is not a file.
    var r4 = c->size("/nope");
    show("size missing", r4); // @output size missing: Could not determine the size of "ftp://ballerina:***@127.0.0.1:PORT/nope" because it is not a file.
    var r5 = c->list("/t.txt");
    show("list file", r5); // @output list file: Could not list the contents of "ftp://ballerina:***@127.0.0.1:PORT/t.txt" because it is not a folder.
    var r6 = c->getBytes("/a");
    show("read dir", r6); // @output read dir: Could not determine the size of "ftp://ballerina:***@127.0.0.1:PORT/a" because it is not a file.
    var r6s = c->getBytesAsStream("/a");
    show("stream dir", r6s); // @output stream dir: Could not read from "ftp://ballerina:***@127.0.0.1:PORT/a" because it is not a file.
    var r7 = c->getText("/nope");
    show("read missing", r7); // @output read missing: Failed to read file: ftp://ballerina:***@127.0.0.1:PORT/nope not found
    var r8 = c->delete("/nope");
    show("delete missing", r8); // @output delete missing: Failed to delete file: ftp://ballerina:***@127.0.0.1:PORT/nope not found
    var r9 = c->rmdir("/nope");
    show("rmdir missing", r9); // @output rmdir missing: Failed to delete directory: ftp://ballerina:***@127.0.0.1:PORT/nope not found
    var r10 = c->mkdir("/a/b");
    show("mkdir exists", r10); // @output mkdir exists: Directory exists: ftp://ballerina:***@127.0.0.1:PORT/a/b
    var r11 = c->rename("/nope", "/x");
    show("rename missing", r11); // @output rename missing: Failed to rename file: ftp://ballerina:***@127.0.0.1:PORT/nope not found
    var r12 = c->rename("/t.txt", "/bad.txt");
    show("rename exists", r12); // @output rename exists: The file at ftp://ballerina:***@127.0.0.1:PORT/bad.txt already exists or it is a directory
    var r13 = c->copy("/nope", "/x");
    show("copy missing", r13); // @output copy missing: Failed to copy file: ftp://ballerina:***@127.0.0.1:PORT/nope not found
    var r14 = c->copy("/t.txt", "/bad.txt");
    show("copy exists", r14); // @output copy exists: The file at ftp://ballerina:***@127.0.0.1:PORT/bad.txt already exists
    var r15 = c->copy("/a", "/z");
    show("copy dir", r15); // @output copy dir: ok
    boolean copiedIsDir = check c->isDirectory("/z");
    ftp:FileInfo[] copiedChildren = check c->list("/z");
    io:println(copiedIsDir, " ", copiedChildren.length()); // @output true 0

    var r16 = c->move("/t.txt", "/moved/deep/t.txt");

    show("move", r16); // @output move: ok
    boolean movedExists = check c->exists("/moved/deep/t.txt");
    io:println(movedExists); // @output true
    var r17 = c->delete("/moved/deep");
    show("delete dir", r17); // @output delete dir: Could not delete "ftp://ballerina:***@127.0.0.1:PORT/moved/deep".
    var r18 = c->rmdir("/moved");
    show("rmdir", r18); // @output rmdir: ok
    boolean gone = check c->exists("/moved");
    io:println(gone); // @output false
    check c->mkdir("/empty/dir");
    var r19 = c->delete("/empty/dir");
    show("delete empty dir", r19); // @output delete empty dir: ok
    var r20 = c->rmdir("/bad.txt");
    show("rmdir file", r20); // @output rmdir file: ok

    ftp:Error? missingErr = c->delete("/nope");
    io:println(missingErr is ftp:FileNotFoundError); // @output true
    ftp:Error? existsErr = c->mkdir("/a");
    io:println(existsErr is ftp:FileAlreadyExistsError); // @output true

    check c->close();
    check c->close();
    var r21 = c->getText("/doc.json");
    show("after close", r21); // @output after close: FTP Client is already closed, hence further operations are not allowed
    var r22 = c->exists("/doc.json");
    show("exists after close", r22); // @output exists after close: FTP Client is already closed, hence further operations are not allowed
}
