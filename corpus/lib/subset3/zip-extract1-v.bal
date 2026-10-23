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

// Entries named `../evil.txt`, `/abs.txt`, `a\b.txt`, `c:drive.txt`, `notes.txt:evil`, `a/./b.txt` and `nul<zero byte>zero.txt`,
// and a symbolic link `ok/link`.
const string UNSAFE = "504b0304140000000800c053cf508316dc8c03000000010000000b0000002e2e2f6576696c2e747874ab0000504b03041400"
    + "00000800c053cf508316dc8c0300000001000000080000002f6162732e747874ab0000504b0304140000000800c053cf5083"
    + "16dc8c030000000100000007000000615c622e747874ab0000504b0304140000000800c053cf508316dc8c03000000010000"
    + "000b000000633a64726976652e747874ab0000504b0304140000000800c053cf508316dc8c03000000010000000e0000006e"
    + "6f7465732e7478743a6576696cab0000504b0304140000000800c053cf508316dc8c030000000100000009000000612f2e2f"
    + "622e747874ab0000504b0304140000000800c053cf508316dc8c03000000010000000c0000006e756c007a65726f2e747874"
    + "ab0000504b0304140000000800c053cf501ebbc1130b00000009000000070000006f6b2f6c696e6bd3d3d3d7d3d34f2d4906"
    + "00504b01021403140000000800c053cf508316dc8c03000000010000000b0000000000000000000000a481000000002e2e2f"
    + "6576696c2e747874504b01021403140000000800c053cf508316dc8c0300000001000000080000000000000000000000a481"
    + "2c0000002f6162732e747874504b01021403140000000800c053cf508316dc8c030000000100000007000000000000000000"
    + "0000a48155000000615c622e747874504b01021403140000000800c053cf508316dc8c03000000010000000b000000000000"
    + "0000000000a4817d000000633a64726976652e747874504b01021403140000000800c053cf508316dc8c0300000001000000"
    + "0e0000000000000000000000a481a90000006e6f7465732e7478743a6576696c504b01021403140000000800c053cf508316"
    + "dc8c0300000001000000090000000000000000000000a481d8000000612f2e2f622e747874504b01021403140000000800c0"
    + "53cf508316dc8c03000000010000000c0000000000000000000000a481020100006e756c007a65726f2e747874504b010214"
    + "03140000000800c053cf501ebbc1130b00000009000000070000000000000000000000ffa12f0100006f6b2f6c696e6b504b"
    + "05060000000008000800bf0100005f0100000000";

// `d/` and `d/f.txt` carry extended timestamps and `g.txt` an NTFS time.
const string STAMPED = "504b0304140000000800c053cf5000000000020000000000000002000900642f5554050001002f68590300504b0304140000"
    + "000800c053cf50cf79e5bb090000000700000007000900642f662e747874555405000100105e5f2b2e49cc2d484d0100504b"
    + "0304140000000800c053cf500773ee53060000000400000005002400672e7478740a002000000000000100180087d67fc647"
    + "17da0100000000000000000000000000000000cb2b492b0600504b01021403140000000800c053cf50000000000200000000"
    + "000000020009000000000000001000ed4100000000642f5554050001002f6859504b01021403140000000800c053cf50cf79"
    + "e5bb0900000007000000070009000000000000000000a4812b000000642f662e747874555405000100105e5f504b01021403"
    + "140000000800c053cf500773ee530600000004000000050024000000000000000000a48162000000672e7478740a00200000"
    + "0000000100180087d67fc64717da0100000000000000000000000000000000504b05060000000003000300ce000000af0000"
    + "000000";

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
    if result is zip:LimitExceededError {
        return "LimitExceededError";
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
    string message = result is error ? scrub(result.message(), dirName) : "";
    io:println(kind(result), ": ", replace(message, "\u{0000}", "<NUL>"));
}

public function main() returns error? {
    string dir = check file:createTempDir(prefix = "bal-zip-extract1-");
    string dirName = check file:basename(dir);
    string archive = check file:joinPath(dir, "plain.zip");
    byte[] big = [];
    foreach int i in 0 ..< 20000 {
        big.push(<byte>(i % 251));
    }
    zip:ArchiveWriter writer = check new (archive, {level: zip:NONE});
    check writer.addEntry("a.txt", "new".toBytes());
    check writer.addEntry("sub/b.txt", "b".toBytes());
    check writer.addEntry("empty/", []);
    check writer.addEntry("big.bin", big);
    check writer.close();

    string out = check file:joinPath(dir, "out");
    check zip:decompress(archive, out);
    io:println(check io:fileReadString(check file:joinPath(out, "sub", "b.txt"))); // @output b
    io:println(check file:test(check file:joinPath(out, "empty"), file:IS_DIR)); // @output true

    string a = check file:joinPath(out, "a.txt");
    check io:fileWriteString(a, "old");
    show(zip:decompress(archive, out), dirName); // @output FileSystemError: cannot extract entry 'a.txt': file '<dir>/out/a.txt' already exists
    check zip:decompress(archive, out, {fileWriteMode: zip:SKIP});
    io:println(check io:fileReadString(a)); // @output old
    check zip:decompress(archive, out, {fileWriteMode: zip:REPLACE});
    io:println(check io:fileReadString(a)); // @output new

    string folderInTheWay = check file:joinPath(dir, "folder");
    check file:createDir(check file:joinPath(folderInTheWay, "a.txt"), file:RECURSIVE);
    show(zip:decompress(archive, folderInTheWay, {fileWriteMode: zip:REPLACE}), dirName);
    // @output FileSystemError: cannot extract entry 'a.txt': a directory already exists at '<dir>/folder/a.txt'
    string fileInTheWay = check file:joinPath(dir, "file");
    check file:createDir(fileInTheWay);
    check io:fileWriteString(check file:joinPath(fileInTheWay, "sub"), "x");
    show(zip:decompress(archive, fileInTheWay), dirName);
    // @output FileSystemError: cannot extract entry 'sub/b.txt': a file already exists at '<dir>/file/sub', where a directory is required

    zip:ArchiveReader reader = check new (archive);
    string single = check file:joinPath(dir, "single.txt");
    check reader.extractEntry("a.txt", single);
    show(reader.extractEntry("a.txt", single), dirName); // @output FileSystemError: cannot extract entry 'a.txt': file '<dir>/single.txt' already exists
    check io:fileWriteString(single, "kept");
    check reader.extractEntry("a.txt", single, {fileWriteMode: zip:SKIP});
    io:println(check io:fileReadString(single)); // @output kept
    check reader.extractEntry("a.txt", single, {fileWriteMode: zip:REPLACE});
    io:println(check io:fileReadString(single)); // @output new
    show(reader.extractEntry("a.txt", dir), dirName); // @output FileSystemError: cannot extract entry 'a.txt': a directory already exists at '<dir>'
    show(reader.extractEntry("a.txt", dir + "//missing//a.txt"), dirName);
    // @output FileSystemError: cannot extract entry 'a.txt': the content could not be written to '<dir>/missing/a.txt'
    check reader.extractEntry("empty/", check file:joinPath(dir, "deep", "er", "empty"));
    io:println(check file:test(check file:joinPath(dir, "deep", "er", "empty"), file:IS_DIR)); // @output true

    show(reader.extractAll(check file:joinPath(dir, "l1"), {limits: {maxEntries: 0}}), dirName);
    // @output Error: invalid 'maxEntries': the value must be positive; omit it to place no limit on the number of entries
    show(reader.extractAll(check file:joinPath(dir, "l1"), {limits: {maxTotalSize: -1}}), dirName);
    // @output Error: invalid 'maxTotalSize': the value must be positive; omit it to place no limit on the total size
    show(reader.extractEntry("a.txt", single, {limits: {maxCompressionRatio: 0}}), dirName);
    // @output Error: invalid 'maxCompressionRatio': the value must be positive; omit it to place no limit on how far an entry may expand
    show(reader.extractAll(check file:joinPath(dir, "l2"), {limits: {maxEntries: 2}}), dirName);
    // @output LimitExceededError: cannot extract the archive: it contains more than the 2 entries allowed
    string l3 = check file:joinPath(dir, "l3");
    show(reader.extractAll(l3, {limits: {maxTotalSize: 10000}}), dirName);
    // @output LimitExceededError: cannot extract entry 'big.bin': the extracted size would exceed the 10000 bytes allowed
    io:println((check file:getMetaData(check file:joinPath(l3, "big.bin"))).size); // @output 8192
    string l4 = check file:joinPath(dir, "l4");
    show(reader.extractAll(l4, {limits: {maxTotalSize: 6}}), dirName);
    // @output LimitExceededError: cannot extract entry 'big.bin': the extracted size would exceed the 6 bytes allowed
    io:println(check file:test(check file:joinPath(l4, "big.bin"), file:EXISTS)); // @output false
    check reader.extractEntry("big.bin", check file:joinPath(dir, "big.bin"), {limits: {maxTotalSize: 6}});
    check reader.close();

    string zeros = check file:joinPath(dir, "zeros.zip");
    zip:ArchiveWriter zerosWriter = check new (zeros);
    byte[] nothing = [];
    foreach int _ in 0 ..< 40000 {
        nothing.push(0);
    }
    check zerosWriter.addEntry("zeros.bin", nothing);
    check zerosWriter.close();
    zip:ArchiveReader zerosReader = check new (zeros);
    show(zerosReader.extractAll(check file:joinPath(dir, "r1"), {limits: {maxCompressionRatio: 2}}), dirName);
    // @output LimitExceededError: cannot extract entry 'zeros.bin': it expands to more than 2 times its compressed size
    show(zerosReader.extractEntry("zeros.bin", check file:joinPath(dir, "r2.bin"), {limits: {maxCompressionRatio: 2}}), dirName);
    // @output LimitExceededError: cannot extract entry 'zeros.bin': it expands to more than 2 times its compressed size
    check zerosReader.extractEntry("zeros.bin", check file:joinPath(dir, "r3.bin"), {limits: {maxCompressionRatio: 100000}});
    io:println((check file:getMetaData(check file:joinPath(dir, "r3.bin"))).size); // @output 40000
    check zerosReader.close();

    string unsafe = check file:joinPath(dir, "unsafe.zip");
    check io:fileWriteBytes(unsafe, check fromHex(UNSAFE));
    zip:ArchiveReader unsafeReader = check new (unsafe);
    foreach zip:Entry entry in check unsafeReader.entries() {
        show(unsafeReader.extractEntry(entry.name, check file:joinPath(dir, "u.txt")), dirName);
    }
    // @output UnsafePathError: invalid entry name '../evil.txt': a name cannot contain a '..' segment
    // @output UnsafePathError: invalid entry name '/abs.txt': a name cannot be absolute
    // @output UnsafePathError: invalid entry name 'a\b.txt': a name cannot contain '\'
    // @output UnsafePathError: invalid entry name 'c:drive.txt': a name cannot start with a drive letter
    // @output UnsafePathError: invalid entry name 'notes.txt:evil': a name cannot contain ':'
    // @output UnsafePathError: invalid entry name 'a/./b.txt': a name cannot contain a '.' segment
    // @output UnsafePathError: invalid entry name 'nul<NUL>zero.txt': a name cannot contain a zero byte
    // @output UnsupportedEntryError: cannot extract entry 'ok/link': symbolic links are not supported
    show(unsafeReader.extractAll(check file:joinPath(dir, "unsafe")), dirName);
    // @output UnsafePathError: invalid entry name '../evil.txt': a name cannot contain a '..' segment
    check unsafeReader.close();

    string stamped = check file:joinPath(dir, "stamped.zip");
    check io:fileWriteBytes(stamped, check fromHex(STAMPED));
    string times = check file:joinPath(dir, "times");
    check zip:decompress(stamped, times);
    io:println((check file:getMetaData(check file:joinPath(times, "d"))).modifiedTime[0]); // @output 1500000000
    io:println((check file:getMetaData(check file:joinPath(times, "d", "f.txt"))).modifiedTime[0]); // @output 1600000000
    io:println((check file:getMetaData(check file:joinPath(times, "g.txt"))).modifiedTime[0]); // @output 1700000000
    zip:ArchiveReader stampedReader = check new (stamped);
    string g = check file:joinPath(dir, "g.txt");
    check stampedReader.extractEntry("g.txt", g);
    io:println((check file:getMetaData(g)).modifiedTime[0]); // @output 1700000000
    check stampedReader.close();
    check file:remove(dir, file:RECURSIVE);
}
