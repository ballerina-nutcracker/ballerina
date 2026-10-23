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

public function main() returns error? {
    string root = check file:createTempDir();
    string src = check file:joinPath(root, "reports");
    check file:createDir(check file:joinPath(src, "sub"), file:RECURSIVE);
    check io:fileWriteString(check file:joinPath(src, "a.txt"), "hello");
    check io:fileWriteString(check file:joinPath(src, "sub", "b.txt"), "world world world world");

    // compress / listEntries
    string archive = check file:joinPath(root, "reports.zip");
    check zip:compress(src, archive);
    foreach zip:Entry e in check zip:listEntries(archive) {
        io:println(e.name, " dir=", e.isDirectory, " size=", e.uncompressedSize);
    }
    // @output reports/ dir=true size=0
    // @output reports/a.txt dir=false size=5
    // @output reports/sub/ dir=true size=0
    // @output reports/sub/b.txt dir=false size=23

    zip:Error? again = zip:compress(src, archive);
    io:println(again is zip:FileSystemError);
    // @output true
    check zip:compress(src, archive, {includeSourceDirectory: false, overwrite: true, level: zip:BEST});
    io:println((check zip:listEntries(archive))[0].name);
    // @output a.txt

    // decompress
    string out = check file:joinPath(root, "out");
    check zip:decompress(archive, out);
    io:println(check io:fileReadString(check file:joinPath(out, "sub", "b.txt")));
    // @output world world world world
    zip:Error? clash = zip:decompress(archive, out);
    io:println(clash is zip:FileSystemError);
    // @output true
    check zip:decompress(archive, out, {fileWriteMode: zip:SKIP});
    check zip:decompress(archive, out, {fileWriteMode: zip:REPLACE});

    // ArchiveWriter
    string built = check file:joinPath(root, "built.zip");
    zip:ArchiveWriter writer = check new (built, {level: zip:DEFAULT});
    check writer.addFile(check file:joinPath(src, "a.txt"));
    check writer.addFile(check file:joinPath(src, "a.txt"), "renamed/a.txt");
    check writer.addDirectory(check file:joinPath(src, "sub"), "nested");
    check writer.addEntry("inline.txt", "inline content".toBytes());
    byte[][] chunks = ["streamed ".toBytes(), "content".toBytes()];
    check writer.addEntry("streamed.txt", chunks.toStream());
    check writer.addEntry("empty/", []);
    zip:ArchiveReader origin = check new (archive);
    check writer.copyEntry(origin, "a.txt");
    check origin.close();
    zip:Error? unsafeName = writer.addEntry("../evil.txt", [1]);
    io:println(unsafeName is zip:UnsafePathError);
    // @output true
    check writer.close();

    // ArchiveReader
    zip:ArchiveReader reader = check new (built);
    foreach zip:Entry e in check reader.entries() {
        io:println(e.name);
    }
    // @output a.txt
    // @output renamed/a.txt
    // @output nested/
    // @output nested/b.txt
    // @output inline.txt
    // @output streamed.txt
    // @output empty/
    // @output a.txt
    io:println(check reader.hasEntry("inline.txt"), " ", check reader.hasEntry("missing.txt"));
    // @output true false
    zip:Entry inline = check reader.getEntry("inline.txt");
    io:println(inline.method, " ", inline.uncompressedSize, " ", inline.crc32);
    // @output DEFLATE 14 1122053085
    zip:Entry|zip:Error missing = reader.getEntry("missing.txt");
    io:println(missing is zip:EntryNotFoundError);
    // @output true

    byte[] bytes = check reader.readEntry("streamed.txt");
    io:println(check string:fromBytes(bytes));
    // @output streamed content
    stream<byte[], zip:Error?> content = check reader.readEntry("nested/b.txt");
    int total = 0;
    record {|byte[] value;|}|zip:Error? chunk = content.next();
    while chunk is record {|byte[] value;|} {
        total += chunk.value.length();
        chunk = content.next();
    }
    io:println(total);
    // @output 23

    string single = check file:joinPath(root, "single.txt");
    check reader.extractEntry("inline.txt", single);
    io:println(check io:fileReadString(single));
    // @output inline content

    string all = check file:joinPath(root, "all");
    zip:Error? limited = reader.extractAll(all, {limits: {maxEntries: 2}});
    io:println(limited is zip:LimitExceededError, " ", (<error>limited).message());
    // @output true cannot extract the archive: it contains more than the 2 entries allowed
    zip:Error? badLimit = reader.extractAll(all, {limits: {maxTotalSize: 0}});
    io:println(badLimit is zip:Error);
    // @output true
    check reader.close();

    zip:ArchiveReader|zip:Error notZip = new (single);
    io:println(notZip is zip:InvalidArchiveError);
    // @output true

    check file:remove(root, file:RECURSIVE);
}
