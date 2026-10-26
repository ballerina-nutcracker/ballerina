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

class FailingStream {
    public isolated function next() returns record {|byte[] value;|}|error? {
        return error("source broken");
    }
}

type Row record {|
    string name;
    int qty;
|};

public function main() returns error? {
    ftp:Client c = check new ({
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}}
    });

    byte[][] parts = [[104, 105], [33], [], [63]];
    check c->putBytesAsStream("/s/bytes.bin", parts.toStream());
    byte[] uploaded = check c->getBytes("/s/bytes.bin");
    io:println(uploaded); // @output [104,105,33]

    byte[][] more = [[1, 2, 3]];
    check c->putBytesAsStream("/s/bytes.bin", more.toStream(), ftp:APPEND);
    uploaded = check c->getBytes("/s/bytes.bin");
    io:println(uploaded); // @output [104,105,33,1,2,3]

    stream<byte[], error?> failing = new (new FailingStream());
    ftp:Error? failed = c->putBytesAsStream("/s/failed.bin", failing);
    io:println(failed is ftp:Error ? failed.message() : "ok"); // @output Iterator error: source broken

    string[][] lines = [["a", "b,c"], ["d", "e"]];
    check c->putCsvAsStream("/s/lines.csv", lines.toStream());
    Row[] rows = [{name: "x", qty: 1}, {name: "y\"z", qty: 2}];
    check c->putCsvAsStream("/s/lines.csv", rows.toStream(), ftp:APPEND);
    string csv = check c->getText("/s/lines.csv");
    io:print(csv); // @output a,"b,c"
    // @output d,e
    // @output name,qty
    // @output x,1
    // @output "y""z",2

    byte[] big = [];
    foreach int i in 0 ..< 20000 {
        big.push(<byte>(i % 256));
    }
    check c->putBytes("/s/big.bin", big);
    stream<byte[], error?> bigStream = check c->getBytesAsStream("/s/big.bin");
    int total = 0;
    int chunks = 0;
    record {|byte[] value;|}? next = check bigStream.next();
    while next is record {|byte[] value;|} {
        total += next.value.length();
        chunks += 1;
        next = check bigStream.next();
    }
    io:println(total, " ", chunks >= 3); // @output 20000 true

    stream<byte[] & readonly, io:Error?> legacy = check c->get("/s/big.bin");
    int legacyTotal = 0;
    record {|byte[] & readonly value;|}? legacyNext = check legacy.next();
    while legacyNext is record {|byte[] & readonly value;|} {
        legacyTotal += legacyNext.value.length();
        legacyNext = check legacy.next();
    }
    check legacy.close();
    io:println(legacyTotal); // @output 20000

    check c->putBytes("/s/empty.bin", []);
    stream<byte[] & readonly, io:Error?> emptyLegacy = check c->get("/s/empty.bin");
    record {|byte[] & readonly value;|}? none = check emptyLegacy.next();
    io:println(none is ()); // @output true
    stream<byte[] & readonly, io:Error?>|ftp:Error missingLegacy = c->get("/s/none.bin");
    io:println(missingLegacy is ftp:FileNotFoundError); // @output true

    check c->put("/s/legacy.txt", "plain");
    check c->append("/s/legacy.txt", " more");
    string legacyText = check c->getText("/s/legacy.txt");
    io:println(legacyText); // @output plain more
    json legacyJson = {k: [1, "v"]};
    check c->put("/s/legacy.json", legacyJson);
    string legacyJsonText = check c->getText("/s/legacy.json");
    io:println(legacyJsonText); // @output {"k":[1, "v"]}
    check c->put("/s/legacy.xml", xml `<a>b</a>`);
    string legacyXmlText = check c->getText("/s/legacy.xml");
    io:println(legacyXmlText); // @output <a>b</a>
    (byte[] & readonly)[] blocks = [[120], [121]];
    check c->put("/s/legacy.bin", blocks.toStream());
    byte[] legacyBin = check c->getBytes("/s/legacy.bin");
    io:println(legacyBin); // @output [120,121]

    check c->put("/s/archive.txt", "zip me", ftp:ZIP);
    byte[] zipped = check c->getBytes("/s/archive.zip");
    io:println(zipped[0], zipped[1], zipped[2], zipped[3]); // @output 807534
    ftp:Error?|error noDot = trap c->put("/s/noext", "x", ftp:ZIP);
    io:println(noDot is error && !(noDot is ftp:Error)); // @output true

    check c->close();
}
