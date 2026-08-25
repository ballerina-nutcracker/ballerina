// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
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

import ballerina/avro;
import ballerina/io;

public function main() returns error? {
    avro:Schema digestSchema = check new (string
        `{"type": "fixed", "name": "Md5", "namespace": "demo", "size": 4}`);

    // fixed is written raw, with no length prefix.
    byte[] toEncode = [9, 8, 7, 6];
    byte[] encoded = check digestSchema.toAvro(toEncode);
    io:println(encoded.length()); // @output 4
    byte[] digest = check digestSchema.fromAvro(encoded);
    io:println(digest.length()); // @output 4
    io:println(digest[0]); // @output 9
    io:println(digest[3]); // @output 6

    // A zero-size fixed is rejected as an invalid schema.
    avro:Schema|error nothingSchema = new (string
        `{"type": "fixed", "name": "Nothing", "size": 0}`);
    io:println(nothingSchema is avro:Error); // @output true

    // The byte count must match the declared size exactly.
    byte[] short = [1, 2];
    byte[]|avro:Error tooShort = digestSchema.toAvro(short);
    io:println(tooShort is avro:Error); // @output true
    byte[] long = [1, 2, 3, 4, 5];
    byte[]|avro:Error tooLong = digestSchema.toAvro(long);
    io:println(tooLong is avro:Error); // @output true

    // A non-list value cannot fill a fixed schema.
    byte[]|avro:Error notBytes = digestSchema.toAvro("abcd");
    io:println(notBytes is avro:Error); // @output true

    // toAvro validates against the value's own type, not just its elements:
    // an int[] is rejected even though every value fits in a byte.
    int[] intsInByteRange = [1, 2, 3, 4];
    byte[]|avro:Error wrongType = digestSchema.toAvro(intsInByteRange);
    io:println(wrongType is avro:Error); // @output true

    // fixed nests inside records and unions like any other type.
    avro:Schema wrapper = check new (string `{
        "type": "record", "name": "Wrapper", "namespace": "demo",
        "fields": [{"name": "hash",
                    "type": ["null", {"type": "fixed", "name": "Md5", "size": 4}]}]}`);
    byte[] hashBytes = [1, 2, 3, 4];
    map<byte[]> wrapped = check wrapper.fromAvro(check wrapper.toAvro({hash: hashBytes}));
    byte[]? hash = wrapped["hash"];
    if hash is byte[] {
        io:println(hash[1]); // @output 2
    }
}
