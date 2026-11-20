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

// Entries recorded on Unix, macOS and FAT; a symbolic link; directories; extended, pre-epoch and NTFS
// timestamps; comments; duplicate names; CP437 and malformed UTF-8 names; an encrypted entry and one
// compressed with bzip2.
const string FIELDS = "504b0304140000000800c053cf503e6a6df2030000000100000008000000756e69782e7478742b0500504b03041400000008"
    + "00c053cf5068f201e10300000001000000070000006d61632e747874cb0500504b0304140000000800c053cf50e02bd37603"
    + "00000001000000070000006661742e7478744b0300504b0304140000000800c053cf50a85a6a8503000000010000000c0000"
    + "00747970656f6e6c792e7478742b0100504b0304140000000800c053cf500bcf0e1b03000000010000000600000073657475"
    + "69642b0600504b0304140000000800c053cf5049baa4000c0000000a000000040000006c696e6b2b492c4a4f2dd12ba92801"
    + "00504b0304140000000800c053cf50000000000200000000000000070000004641544449525c0300504b0304140000000800"
    + "c053cf5000000000020000000000000006000000646f736469720300504b0304140000000800c053cf5043beb7e803000000"
    + "010000000600090074732e747874555405000100105e5f4b0400504b0304140000000800c053cf50f9efbe71030000000100"
    + "0000070009006e65672e74787455540500019cffffff4b0200504b0304140000000800c053cf506fdfb90603000000010000"
    + "00080024006e7466732e7478740a002000000000000100180087d67fc64717da01000000000000000000000000000000004b"
    + "0600504b0304140000000000c053cf501d73ee6d0e0000000e0000000a00000073746f7265642e74787473746f7265642063"
    + "6f6e74656e74504b0304140000000800c053cf5057ee71920700000005000000070000006475702e7478744bcb2c2a2e0100"
    + "504b0304140000000800c053cf5069111fb60800000006000000070000006475702e7478742b4e4dcecf4b0100504b030414"
    + "0000000800c053cf506fd73b1c070000000500000008000000636166822e7478744b2e3031360700504b0304140000080800"
    + "c053cf50813bf835060000000400000009000000626164e2822e7478742b2d49b30000504b0304140001000800c053cf50e5"
    + "e8a25c080000000600000007000000656e632e7478742b4e4d2e4a2d0100504b0304140000000c00c053cf50203558d90700"
    + "00000500000008000000627a69702e747874cb2fc9482d0200504b0304140000000800c053cf500000000002000000000000"
    + "00040000006469722f0300504b01021403140000000800c053cf503e6a6df203000000010000000800000005000000000000"
    + "00a48100000000756e69782e747874706c61696e504b01021413140000000800c053cf5068f201e103000000010000000700"
    + "000000000000000000008081290000006d61632e747874504b01021400140000000800c053cf50e02bd37603000000010000"
    + "00070000000000000000000000a481510000006661742e747874504b01021403140000000800c053cf50a85a6a8503000000"
    + "010000000c0000000000000000000000008079000000747970656f6e6c792e747874504b01021403140000000800c053cf50"
    + "0bcf0e1b0300000001000000060000000000000000000000ed89a6000000736574756964504b01021403140000000800c053"
    + "cf5049baa4000c0000000a000000040000000000000000000000ffa1cd0000006c696e6b504b01021400140000000800c053"
    + "cf500000000002000000000000000700000000000000000000008001fb0000004641544449525c504b010214001400000008"
    + "00c053cf50000000000200000000000000060000000000000000001000000022010000646f73646972504b01021403140000"
    + "000800c053cf5043beb7e80300000001000000060009000000000000000000a4814801000074732e74787455540500010010"
    + "5e5f504b01021403140000000800c053cf50f9efbe710300000001000000070009000000000000000000a481780100006e65"
    + "672e74787455540500019cffffff504b01021403140000000800c053cf506fdfb90603000000010000000800240005000000"
    + "00000000a481a90100006e7466732e7478740a002000000000000100180087d67fc64717da01000000000000000000000000"
    + "00000000626164e282504b01021403140000000000c053cf501d73ee6d0e0000000e0000000a0000000000000000000000a4"
    + "81f601000073746f7265642e747874504b01021403140000000800c053cf5057ee7192070000000500000007000000000000"
    + "0000000000a4812c0200006475702e747874504b01021403140000000800c053cf5069111fb6080000000600000007000000"
    + "0000000000000000a481580200006475702e747874504b01021400140000000800c053cf506fd73b1c070000000500000008"
    + "0000000000000000000000800185020000636166822e747874504b01021403140000080800c053cf50813bf8350600000004"
    + "000000090000000000000000000000a481b2020000626164e2822e747874504b01021403140001000800c053cf50e5e8a25c"
    + "0800000006000000070000000000000000000000a481df020000656e632e747874504b01021403140000000c00c053cf5020"
    + "3558d90700000005000000080000000000000000000000a4810c030000627a69702e747874504b01021403140000000800c0"
    + "53cf50000000000200000000000000040000000000000000001000ed41390300006469722f504b0506000000001300130034"
    + "0400005d0300000000";

function fromHex(string hex) returns byte[]|error {
    byte[] bytes = [];
    int i = 0;
    while i < hex.length() {
        bytes.push(<byte>check int:fromHexString(hex.substring(i, i + 2)));
        i += 2;
    }
    return bytes;
}

function describe(zip:Entry entry) returns string {
    int? mode = entry?.unixMode;
    string? comment = entry?.comment;
    return string `${entry.name} dir=${entry.isDirectory} link=${entry.isSymlink} ` +
        string `mode=${mode is int ? mode.toString() : "absent"} method=${entry.method} ` +
        string `comment=${comment is string ? comment : "absent"}`;
}

public function main() returns error? {
    string dir = check file:createTempDir(prefix = "bal-zip-reader1-");
    string path = check file:joinPath(dir, "fields.zip");
    check io:fileWriteBytes(path, check fromHex(FIELDS));

    zip:Entry[] entries = check zip:listEntries(path);
    io:println(entries.length()); // @output 19
    foreach zip:Entry entry in entries {
        io:println(describe(entry));
    }
    // @output unix.txt dir=false link=false mode=420 method=DEFLATE comment=plain
    // @output mac.txt dir=false link=false mode=384 method=DEFLATE comment=absent
    // @output fat.txt dir=false link=false mode=absent method=DEFLATE comment=absent
    // @output typeonly.txt dir=false link=false mode=0 method=DEFLATE comment=absent
    // @output setuid dir=false link=false mode=2541 method=DEFLATE comment=absent
    // @output link dir=false link=true mode=511 method=DEFLATE comment=absent
    // @output FATDIR\ dir=true link=false mode=absent method=DEFLATE comment=absent
    // @output dosdir dir=false link=false mode=absent method=DEFLATE comment=absent
    // @output ts.txt dir=false link=false mode=420 method=DEFLATE comment=absent
    // @output neg.txt dir=false link=false mode=420 method=DEFLATE comment=absent
    // @output ntfs.txt dir=false link=false mode=420 method=DEFLATE comment=bad?
    // @output stored.txt dir=false link=false mode=420 method=STORE comment=absent
    // @output dup.txt dir=false link=false mode=420 method=DEFLATE comment=absent
    // @output dup.txt dir=false link=false mode=420 method=DEFLATE comment=absent
    // @output café.txt dir=false link=false mode=absent method=DEFLATE comment=absent
    // @output bad�.txt dir=false link=false mode=420 method=DEFLATE comment=absent
    // @output enc.txt dir=false link=false mode=420 method=DEFLATE comment=absent
    // @output bzip.txt dir=false link=false mode=420 method=OTHER comment=absent
    // @output dir/ dir=true link=false mode=493 method=DEFLATE comment=absent

    io:println(entries[8].modifiedTime); // @output [1600000000,0]
    io:println(entries[9].modifiedTime); // @output [0,0]
    io:println(entries[10].modifiedTime); // @output [1700000000,0.123456700]
    zip:Entry stored = entries[11];
    io:println(stored.uncompressedSize, " ", stored.compressedSize, " ", stored.crc32); // @output 14 14 1844343581

    zip:ArchiveReader reader = check new (path);
    io:println(check reader.hasEntry("dup.txt"), " ", check reader.hasEntry("DUP.TXT")); // @output true false
    byte[] first = check reader.readEntry("dup.txt");
    io:println(check string:fromBytes(first)); // @output first
    byte[] target = check reader.readEntry("link");
    io:println(check string:fromBytes(target)); // @output target.txt
    byte[] folder = check reader.readEntry("dir/");
    io:println(folder.length()); // @output 0
    byte[] cp437 = check reader.readEntry("café.txt");
    io:println(check string:fromBytes(cp437)); // @output cp437
    zip:Entry ts = check reader.getEntry("ts.txt");
    io:println(ts.name, " ", ts.uncompressedSize); // @output ts.txt 1

    byte[]|zip:Error encrypted = reader.readEntry("enc.txt");
    io:println(encrypted is zip:UnsupportedEntryError); // @output true
    if encrypted is zip:Error {
        io:println(encrypted.message()); // @output cannot read entry 'enc.txt': encrypted entries are not supported
    }
    stream<byte[], zip:Error?>|zip:Error other = reader.readEntry("bzip.txt");
    io:println(other is zip:UnsupportedEntryError); // @output true
    if other is zip:Error {
        io:println(other.message()); // @output entry 'bzip.txt' uses a compression method that is not supported
    }
    zip:Entry|zip:Error missing = reader.getEntry("missing.txt");
    io:println(missing is zip:EntryNotFoundError); // @output true
    if missing is zip:Error {
        io:println(missing.message()); // @output the archive contains no entry named 'missing.txt'
    }
    byte[]|zip:Error notThere = reader.readEntry("missing.txt");
    io:println(notThere is zip:EntryNotFoundError); // @output true
    check reader.close();
    check file:remove(dir, file:RECURSIVE);
}
