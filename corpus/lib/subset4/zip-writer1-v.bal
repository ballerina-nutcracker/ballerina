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

// `enc.txt` is encrypted, `bzip.txt` is compressed with bzip2 and carries a comment, and `café.txt` has a
// CP437 name and an extended timestamp.
const string COPY = "504b0304140001000800c053cf50e5e8a25c080000000600000007000000656e632e7478742b4e4d2e4a2d0100504b030414"
    + "0000000c00c053cf50203558d9070000000500000008000000627a69702e747874cb2fc9482d0200504b0304140000000800"
    + "c053cf506fd73b1c070000000500000008000900636166822e747874555405000100105e5f4b2e3031360700504b01021403"
    + "140001000800c053cf50e5e8a25c0800000006000000070000000000000000000000a48100000000656e632e747874504b01"
    + "021403140000000c00c053cf50203558d90700000005000000080000000400000000000000a4812d000000627a69702e7478"
    + "746e6f7465504b01021400140000000800c053cf506fd73b1c070000000500000008000900000000000000000080015a0000"
    + "00636166822e747874555405000100105e5f504b05060000000003000300ae000000900000000000";

function fromHex(string hex) returns byte[]|error {
    byte[] bytes = [];
    int i = 0;
    while i < hex.length() {
        bytes.push(<byte>check int:fromHexString(hex.substring(i, i + 2)));
        i += 2;
    }
    return bytes;
}

function replace(string text, string needle, string replacement) returns string {
    int? at = text.indexOf(needle);
    if at is () {
        return text;
    }
    return text.substring(0, at) + replacement + replace(text.substring(at + needle.length()), needle, replacement);
}

// Shows each quoted path of a message relative to the test directory, with `/` as the separator, so that
// the output is the same whatever the platform and however the temporary directory resolves.
function scrub(string text, string dirName) returns string {
    string result = "";
    string rest = text;
    while true {
        int? open = rest.indexOf("'");
        if open is () {
            return result + rest;
        }
        int? close = rest.indexOf("'", open + 1);
        if close is () {
            return result + rest;
        }
        string quoted = rest.substring(open + 1, close);
        int? at = quoted.indexOf(dirName);
        if at is int {
            quoted = "<dir>" + replace(quoted.substring(at + dirName.length()), "\\", "/");
        }
        result += rest.substring(0, open + 1) + quoted + "'";
        rest = rest.substring(close + 1);
    }
}

function kind(error? result) returns string {
    if result is () {
        return "ok";
    }
    if result is zip:UnsafePathError {
        return "UnsafePathError";
    }
    if result is zip:InvalidArchiveError {
        return "InvalidArchiveError";
    }
    if result is zip:EntryNotFoundError {
        return "EntryNotFoundError";
    }
    if result is zip:UnsupportedEntryError {
        return "UnsupportedEntryError";
    }
    if result is zip:FileSystemError {
        return "FileSystemError";
    }
    return "Error";
}

function show(error? result, string dirName) {
    io:println(kind(result), ": ", result is error ? replace(scrub(result.message(), dirName), "\u{0000}", "<NUL>") : "");
}

isolated class Chunks {
    private final string[] & readonly chunks;
    private final boolean failRead;
    private final boolean failClose;
    private int index = 0;

    isolated function init(string[] & readonly chunks, boolean failRead = false, boolean failClose = false) {
        self.chunks = chunks;
        self.failRead = failRead;
        self.failClose = failClose;
    }

    public isolated function next() returns record {|byte[] value;|}|error? {
        int current;
        lock {
            current = self.index;
            self.index += 1;
        }
        if current >= self.chunks.length() {
            return self.failRead ? error("read failed") : ();
        }
        // Built outside the lock: the interpreter does not take this call as an isolated expression
        // (ballerina-nutcracker/ballerina#1429).
        return {value: self.chunks[current].toBytes()};
    }

    public isolated function close() returns error? {
        return self.failClose ? error("close failed") : ();
    }
}

function names(string path) returns string[]|error {
    return from zip:Entry entry in check zip:listEntries(path)
        select entry.name;
}

public function main() returns error? {
    string dir = check file:createTempDir(prefix = "bal-zip-writer1-");
    string dirName = check file:basename(dir);
    string src = check file:joinPath(dir, "src");
    check file:createDir(check file:joinPath(src, "b", "deep"), file:RECURSIVE);
    check file:createDir(check file:joinPath(src, "a"));
    check file:createDir(check file:joinPath(dir, "hollow"));
    check io:fileWriteString(check file:joinPath(src, "z.txt"), "zed");
    check io:fileWriteString(check file:joinPath(src, "b", "one.txt"), "one");
    check io:fileWriteString(check file:joinPath(src, "b", "deep", "two.txt"), "two two");
    string zText = check file:joinPath(src, "z.txt");

    string built = check file:joinPath(dir, "built.zip");
    zip:ArchiveWriter writer = check new (built);
    check writer.addFile(zText);
    check writer.addFile(zText, "renamed/z.txt");
    check writer.addDirectory(check file:joinPath(src, "b"));
    check writer.addDirectory(check file:joinPath(src, "b"), "top/");
    check writer.addDirectory(check file:joinPath(dir, "hollow"));
    check writer.addEntry("bytes.txt", "bytes".toBytes());
    check writer.addEntry("empty.txt", []);
    check writer.addEntry("folder/", []);
    check writer.addEntry("streamed.txt", new stream<byte[], error?>(new Chunks(["ab", "", "cd"])));
    check writer.addEntry("streamed-folder/", new stream<byte[], error?>(new Chunks(["", ""])));
    show(writer.addFile(check file:joinPath(src, "missing.txt")), dirName);
    // @output FileSystemError: cannot add file '<dir>/src/missing.txt': no file exists at that path
    show(writer.addFile(src), dirName);
    // @output FileSystemError: cannot add file '<dir>/src': it is a directory; use 'addDirectory' instead
    show(writer.addFile(zText, "dir/"), dirName);
    // @output Error: cannot add entry 'dir/': a name ending in '/' denotes a directory and cannot hold file content
    show(writer.addFile(zText, ""), dirName); // @output UnsafePathError: invalid entry name: the name is empty
    show(writer.addFile(zText, "a\\b"), dirName); // @output UnsafePathError: invalid entry name 'a\b': a name cannot contain '\'
    show(writer.addFile(zText, "nul\u{0000}"), dirName); // @output UnsafePathError: invalid entry name 'nul<NUL>': a name cannot contain a zero byte
    show(writer.addDirectory(zText), dirName);
    // @output FileSystemError: cannot add directory '<dir>/src/z.txt': no directory exists at that path
    show(writer.addDirectory(src, "../up"), dirName); // @output UnsafePathError: invalid entry name '../up': a name cannot contain a '..' segment
    show(writer.addEntry("full/", [1]), dirName);
    // @output Error: cannot add entry 'full/': a name ending in '/' denotes a directory, which holds no content
    show(writer.addEntry("full/", new stream<byte[], error?>(new Chunks(["", "x"]))), dirName);
    // @output Error: cannot add entry 'full/': a name ending in '/' denotes a directory, which holds no content
    show(writer.addEntry("bad-read.txt", new stream<byte[], error?>(new Chunks(["x"], failRead = true))), dirName);
    // @output Error: cannot add entry 'bad-read.txt': the content stream could not be read
    show(writer.addEntry("bad-first.txt", new stream<byte[], error?>(new Chunks([], failRead = true))), dirName);
    // @output Error: cannot add entry 'bad-first.txt': the content stream could not be read
    show(writer.addEntry("bad-dir/", new stream<byte[], error?>(new Chunks([], failRead = true))), dirName);
    // @output Error: cannot add entry 'bad-dir/': the content stream could not be read
    show(writer.addEntry("bad-close.txt", new stream<byte[], error?>(new Chunks(["y"], failClose = true))), dirName);
    // @output Error: cannot add entry 'bad-close.txt': the content stream could not be closed

    string copyPath = check file:joinPath(dir, "copy.zip");
    check io:fileWriteBytes(copyPath, check fromHex(COPY));
    zip:ArchiveReader copySource = check new (copyPath);
    check writer.copyEntry(copySource, "bzip.txt");
    check writer.copyEntry(copySource, "café.txt");
    show(writer.copyEntry(copySource, "enc.txt"), dirName);
    // @output UnsupportedEntryError: cannot copy entry 'enc.txt': encrypted entries cannot be written into another archive
    show(writer.copyEntry(copySource, "missing.txt"), dirName);
    // @output EntryNotFoundError: the source archive contains no entry named 'missing.txt'
    show(writer.copyEntry(copySource, "../x"), dirName); // @output UnsafePathError: invalid entry name '../x': a name cannot contain a '..' segment
    check copySource.close();
    show(writer.copyEntry(copySource, "bzip.txt"), dirName);
    // @output InvalidArchiveError: cannot copy entry 'bzip.txt': the source archive is closed
    check writer.close();
    check writer.close();
    show(writer.addEntry("late.txt", [1]), dirName); // @output InvalidArchiveError: cannot write to the archive: it is closed
    show(writer.addFile(zText), dirName); // @output InvalidArchiveError: cannot write to the archive: it is closed
    show(writer.addDirectory(src), dirName); // @output InvalidArchiveError: cannot write to the archive: it is closed

    io:println(check names(built));
    // @output ["z.txt","renamed/z.txt","b/","b/deep/","b/deep/two.txt","b/one.txt","top/","top/deep/","top/deep/two.txt","top/one.txt","hollow/","bytes.txt","empty.txt","folder/","streamed.txt","streamed-folder/","bad-read.txt","bad-close.txt","bzip.txt","café.txt"]
    zip:ArchiveReader reader = check new (built);
    foreach string name in ["z.txt", "top/deep/two.txt", "bytes.txt", "empty.txt", "streamed.txt", "bad-read.txt", "bad-close.txt"] {
        byte[] content = check reader.readEntry(name);
        io:println(name, "=", check string:fromBytes(content));
    }
    // @output z.txt=zed
    // @output top/deep/two.txt=two two
    // @output bytes.txt=bytes
    // @output empty.txt=
    // @output streamed.txt=abcd
    // @output bad-read.txt=x
    // @output bad-close.txt=y
    zip:Entry folder = check reader.getEntry("folder/");
    io:println(folder.isDirectory, " ", folder.method, " ", folder?.unixMode is ()); // @output true STORE true
    zip:Entry bytesEntry = check reader.getEntry("bytes.txt");
    io:println(bytesEntry.method, " ", bytesEntry.uncompressedSize, " ", bytesEntry.crc32); // @output DEFLATE 5 2979611598
    zip:Entry copied = check reader.getEntry("bzip.txt");
    io:println(copied.method, " ", copied?.comment); // @output OTHER note
    byte[]|zip:Error stillOther = reader.readEntry("bzip.txt");
    io:println(stillOther is zip:UnsupportedEntryError); // @output true
    zip:Entry cafe = check reader.getEntry("café.txt");
    io:println(cafe.modifiedTime, " ", check string:fromBytes(check reader.readEntry("café.txt"))); // @output [1600000000,0] cp437
    check reader.close();

    string stored = check file:joinPath(dir, "stored.zip");
    zip:ArchiveWriter storeWriter = check new (stored, {level: zip:NONE});
    check storeWriter.addFile(zText);
    check storeWriter.addEntry("s.txt", "stored".toBytes());
    check storeWriter.close();
    foreach zip:Entry entry in check zip:listEntries(stored) {
        io:println(entry.name, " ", entry.method, " ", entry.compressedSize); // @output z.txt STORE 3
    }
    // @output s.txt STORE 6
    zip:CompressionLevel[] levels = [zip:FASTEST, zip:DEFAULT, zip:BEST];
    foreach zip:CompressionLevel level in levels {
        string leveled = check file:joinPath(dir, "level.zip");
        zip:ArchiveWriter levelWriter = check new (leveled, {level, overwrite: true});
        check levelWriter.addEntry("l.txt", "level level level".toBytes());
        check levelWriter.close();
        zip:ArchiveReader levelReader = check new (leveled);
        io:println(level, " ", check string:fromBytes(check levelReader.readEntry("l.txt")));
        check levelReader.close();
    }
    // @output FASTEST level level level
    // @output DEFAULT level level level
    // @output BEST level level level

    zip:ArchiveWriter|zip:Error exists = new (built);
    show(exists is error ? exists : (), dirName);
    // @output FileSystemError: cannot create the archive '<dir>/built.zip': a file already exists there; set 'overwrite' to replace it
    zip:ArchiveWriter|zip:Error onDir = new (src, {overwrite: true});
    show(onDir is error ? onDir : (), dirName);
    // @output FileSystemError: cannot create the archive '<dir>/src': a directory already exists at that path
    zip:ArchiveWriter|zip:Error noParent = new (check file:joinPath(dir, "missing", "x.zip"));
    show(noParent is error ? noParent : (), dirName);
    // @output FileSystemError: cannot create the archive '<dir>/missing/x.zip': the file could not be created
    zip:ArchiveWriter|zip:Error badPath = new (dir + "/bad\u{0000}.zip");
    show(badPath is error ? badPath : (), dirName);
    // @output FileSystemError: invalid path '<dir>/bad<NUL>.zip': the platform cannot represent it

    string single = check file:joinPath(dir, "single.zip");
    check zip:compress(zText, single);
    io:println(check names(single)); // @output ["z.txt"]
    check zip:compress(src, single, {overwrite: true, includeSourceDirectory: false});
    io:println(check names(single)); // @output ["a/","b/","b/deep/","b/deep/two.txt","b/one.txt","z.txt"]
    show(zip:compress(src, check file:joinPath(src, "inside.zip")), dirName);
    // @output FileSystemError: cannot create the archive '<dir>/src/inside.zip': it is inside the source directory '<dir>/src'
    show(zip:compress(zText, zText, {overwrite: true}), dirName);
    // @output FileSystemError: cannot create the archive '<dir>/src/z.txt': it is the same file as the source '<dir>/src/z.txt'
    show(zip:compress(check file:joinPath(dir, "nothing"), single), dirName);
    // @output FileSystemError: cannot resolve the path '<dir>/nothing'
    show(zip:compress(src, check file:joinPath(dir, "nowhere", "x.zip")), dirName);
    // @output FileSystemError: cannot resolve the path '<dir>/nowhere'
    io:println(check io:fileReadString(zText)); // @output zed
    check file:remove(dir, file:RECURSIVE);
}
