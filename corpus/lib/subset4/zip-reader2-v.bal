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

import ballerina/file;
import ballerina/io;
import ballerina/zip;

// The second entry has two bytes of its deflate stream flipped.
const string CORRUPT = "504b0304140000000800c053cf50c2238f5f0e0000000c00000008000000676f6f642e7478744bcfcf4f5148cecf2b49cd2b"
    + "0100504b0304140000000800c053cf50f6d253380b0000001d0000000a00000062726f6b656e2e74787434b7cdc9c957c8c0"
    + "4e0200504b01021403140000000800c053cf50c2238f5f0e0000000c000000080000000000000000000000a4810000000067"
    + "6f6f642e747874504b01021403140000000800c053cf50f6d253380b0000001d0000000a0000000000000000000000a48134"
    + "00000062726f6b656e2e747874504b050600000000020002006e000000670000000000";

// A single entry recorded with Zip64 sizes.
const string ZIP64 = "504b03042d0000000800c053cf506ab6e948ffffffffffffffff0900140062696736342e747874010010000d000000000000"
    + "000f00000000000000abca2c30335148cecf2b49cd2b0100504b01022d032d0000000800c053cf506ab6e9480f0000000d00"
    + "0000090000000000000000000000a4810000000062696736342e747874504b05060000000001000100370000004a00000000"
    + "00";

// The central directory offset points past the end of the file.
const string BADOFFSET = "504b0304140000000800c053cf5043beb7e8030000000100000005000000612e7478744b0400504b01021403140000000800"
    + "c053cf5043beb7e80300000001000000050000000000000000000000a48100000000612e747874504b050600000000010001"
    + "0033000000ffffff7f0000";

// The local header offset of the entry points past the end of the file.
const string BADLOCAL = "504b0304140000000800c053cf5043beb7e8030000000100000005000000612e7478744b0400504b01021403140000000800"
    + "c053cf5043beb7e80300000001000000050000000000000000000000a481ffff7f00612e747874504b050600000000010001"
    + "0033000000260000000000";

function fromHex(string hex) returns byte[]|error {
    byte[] bytes = [];
    int i = 0;
    while i < hex.length() {
        bytes.push(<byte>check int:fromHexString(hex.substring(i, i + 2)));
        i += 2;
    }
    return bytes;
}

function message(any|error result) returns string {
    return result is error ? result.message() : "ok";
}

function chunkLength(record {|byte[] value;|}|zip:Error? chunk) returns string {
    if chunk is record {|byte[] value;|} {
        return chunk.value.length().toString();
    }
    return chunk is () ? "end" : chunk.message();
}

public function main() returns error? {
    string dir = check file:createTempDir(prefix = "bal-zip-reader2-");
    string archive = check file:joinPath(dir, "chunks.zip");
    byte[] big = [];
    foreach int i in 0 ..< 70000 {
        big.push(<byte>(i % 251));
    }
    zip:ArchiveWriter writer = check new (archive, {level: zip:NONE});
    check writer.addEntry("big.bin", big);
    check writer.addEntry("small.txt", "small".toBytes());
    check writer.close();

    zip:ArchiveReader reader = check new (archive);
    stream<byte[], zip:Error?> chunks = check reader.readEntry("big.bin");
    stream<byte[], zip:Error?> other = check reader.readEntry("small.txt");
    io:println(chunkLength(chunks.next())); // @output 65536
    io:println(chunkLength(other.next())); // @output 5
    io:println(chunkLength(chunks.next())); // @output 4464
    io:println(chunkLength(chunks.next())); // @output end
    io:println(chunkLength(chunks.next())); // @output cannot read the entry: the stream is closed
    check chunks.close();

    stream<byte[], zip:Error?> closedEarly = check reader.readEntry("big.bin");
    check closedEarly.close();
    io:println(chunkLength(closedEarly.next())); // @output cannot read the entry: the stream is closed

    stream<byte[], zip:Error?> open = check reader.readEntry("big.bin");
    check reader.close();
    io:println(chunkLength(open.next())); // @output cannot read the entry: the stream is closed
    io:println(chunkLength(other.next())); // @output cannot read the entry: the stream is closed
    check reader.close();

    zip:Entry[]|zip:Error entries = reader.entries();
    io:println(entries is zip:InvalidArchiveError); // @output true
    if entries is zip:Error {
        io:println(entries.message()); // @output cannot read the archive: it is closed
    }
    io:println(message(reader.hasEntry("big.bin"))); // @output cannot read the archive: it is closed
    io:println(message(reader.getEntry("big.bin"))); // @output cannot read the archive: it is closed
    byte[]|zip:Error afterClose = reader.readEntry("big.bin");
    io:println(message(afterClose)); // @output cannot read the archive: it is closed
    io:println(message(reader.extractEntry("big.bin", check file:joinPath(dir, "x")))); // @output cannot read the archive: it is closed
    io:println(message(reader.extractAll(check file:joinPath(dir, "y")))); // @output cannot read the archive: it is closed

    string corrupt = check file:joinPath(dir, "corrupt.zip");
    check io:fileWriteBytes(corrupt, check fromHex(CORRUPT));
    zip:ArchiveReader damaged = check new (corrupt);
    byte[] good = check damaged.readEntry("good.txt");
    io:println(check string:fromBytes(good)); // @output good content
    byte[]|zip:Error broken = damaged.readEntry("broken.txt");
    io:println(broken is zip:InvalidArchiveError, " ", message(broken)); // @output true the content of entry 'broken.txt' could not be read
    stream<byte[], zip:Error?> brokenChunks = check damaged.readEntry("broken.txt");
    io:println(chunkLength(brokenChunks.next())); // @output cannot read the content of the entry
    check brokenChunks.close();
    zip:Error? extracted = damaged.extractEntry("broken.txt", check file:joinPath(dir, "broken.txt"));
    io:println(extracted is zip:InvalidArchiveError, " ", message(extracted)); // @output true the content of entry 'broken.txt' could not be read
    io:println(check file:test(check file:joinPath(dir, "broken.txt"), file:EXISTS)); // @output false
    check damaged.close();

    string zip64 = check file:joinPath(dir, "zip64.zip");
    check io:fileWriteBytes(zip64, check fromHex(ZIP64));
    zip:Entry[] large = check zip:listEntries(zip64);
    io:println(large[0].name, " ", large[0].uncompressedSize); // @output big64.txt 13
    zip:ArchiveReader zip64Reader = check new (zip64);
    byte[] zip64Content = check zip64Reader.readEntry("big64.txt");
    io:println(check string:fromBytes(zip64Content)); // @output zip64 content
    check zip64Reader.close();

    string empty = check file:joinPath(dir, "empty.zip");
    zip:ArchiveWriter emptyWriter = check new (empty);
    check emptyWriter.close();
    io:println((check zip:listEntries(empty)).length()); // @output 0

    string text = check file:joinPath(dir, "notes.txt");
    check io:fileWriteString(text, "not an archive");
    string badOffset = check file:joinPath(dir, "badoffset.zip");
    check io:fileWriteBytes(badOffset, check fromHex(BADOFFSET));
    string badLocal = check file:joinPath(dir, "badlocal.zip");
    check io:fileWriteBytes(badLocal, check fromHex(BADLOCAL));
    foreach string path in [check file:joinPath(dir, "missing.zip"), dir, text, badOffset, badLocal] {
        zip:Entry[]|zip:Error result = zip:listEntries(path);
        string kind = result is zip:FileSystemError ? "FileSystemError" : result is zip:InvalidArchiveError
            ? "InvalidArchiveError" : "other";
        io:println(kind, ": ", message(result).substring(message(result).indexOf("':") ?: 0));
    }
    // @output FileSystemError: ': no file exists at that path
    // @output FileSystemError: ': it is a directory
    // @output InvalidArchiveError: ': it is not a valid ZIP archive
    // @output InvalidArchiveError: ': it is not a valid ZIP archive
    // @output InvalidArchiveError: ': it is not a valid ZIP archive
    check file:remove(dir, file:RECURSIVE);
}
