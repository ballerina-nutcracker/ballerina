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

import ballerina/crypto;
import ballerina/lang.'int as ints;

// Represents UUID module related errors.
public type Error error;

// String representing the nil UUID.
const string NIL_UUID = "00000000-0000-0000-0000-000000000000";

// Represents a UUID.
// + timeLow - The low field of the timestamp
// + timeMid - The middle field of the timestamp
// + timeHiAndVersion - The high field of the timestamp multiplexed with the version number
// + clockSeqHiAndReserved - The high field of the clock sequence multiplexed with the variant
// + clockSeqLo - The low field of the clock sequence
// + node - The spatially unique node identifier
public type Uuid readonly & record {
    ints:Unsigned32 timeLow;
    ints:Unsigned16 timeMid;
    ints:Unsigned16 timeHiAndVersion;
    ints:Unsigned8 clockSeqHiAndReserved;
    ints:Unsigned8 clockSeqLo;
    int node; // Should be Unsigned48, but not available in lang.int at the moment
};

// Represents the UUID versions.
// V1 - UUID generated using the MAC address of the computer and the time of generation
// V3 - UUID generated using MD5 hashing and application-provided text string
// V4 - UUID generated using a pseudo-random number generator
// V5 - UUID generated using SHA-1 hashing and application-provided text string
public enum Version {
    V1,
    V3,
    V4,
    V5
}

// Represents UUIDs strings of well known namespace IDs.
// NAME_SPACE_DNS - Namespace is a fully-qualified domain name
// NAME_SPACE_URL - Namespace is a URL
// NAME_SPACE_OID - Namespace is an ISO OID
// NAME_SPACE_X500 - Namespace is an X.500 DN (in DER or a text output format)
// NAME_SPACE_NIL - Empty UUID
public enum NamespaceUUID {
    NAME_SPACE_DNS = "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
    NAME_SPACE_URL = "6ba7b811-9dad-11d1-80b4-00c04fd430c8",
    NAME_SPACE_OID = "6ba7b812-9dad-11d1-80b4-00c04fd430c8",
    NAME_SPACE_X500 = "6ba7b814-9dad-11d1-80b4-00c04fd430c8",
    NAME_SPACE_NIL = NIL_UUID
}

# Returns a UUID of type 1 as a string.
# ```ballerina
# string uuid1 = uuid:createType1AsString();
# ```
#
# + return - UUID of type 1 as a string
public isolated function createType1AsString() returns string {
    return externCreateType1AsString();
}

# Returns a UUID of type 1 as a UUID record.
# ```ballerina
# uuid:Uuid uuid1 = check uuid:createType1AsRecord();
# ```
#
# + return - UUID of type 1 as a UUID record, or else a `uuid:Error`
public isolated function createType1AsRecord() returns Uuid|Error {
    return check toRecord(createType1AsString());
}

# Returns a UUID of type 3 as a string.
# ```ballerina
# string uuid3 = check uuid:createType3AsString(uuid:NAME_SPACE_DNS, "ballerina.io");
# ```
#
# + namespace - String representation for a pre-defined namespace UUID
# + name - A name within the namespace
#
# + return - UUID of type 3 as a string, or else a `uuid:Error`
public isolated function createType3AsString(NamespaceUUID namespace, string name) returns string|Error {
    string trimmedName = name.trim();
    if trimmedName.length() == 0 {
        return error Error("Name cannot be empty");
    }
    byte[] namespaceBytes = check getBytesFromUuid(namespace);
    byte[] nameBytes = trimmedName.toBytes();
    foreach byte b in nameBytes {
        namespaceBytes.push(b);
    }

    byte[] uuid3 = crypto:hashMd5(namespaceBytes);

    uuid3[6] = uuid3[6] & 0x0f;
    uuid3[6] = <byte>(uuid3[6] | 0x30);
    uuid3[8] = uuid3[8] & 0x3f;
    uuid3[8] = <byte>(uuid3[8] | 0x80);
    return getUuidFromBytes(uuid3);
}

# Returns a UUID of type 3 as a UUID record.
# ```ballerina
# uuid:Uuid uuid3 = check uuid:createType3AsRecord(uuid:NAME_SPACE_DNS, "ballerina.io");
# ```
#
# + namespace - String representation for a pre-defined namespace UUID
# + name - A name within the namespace
#
# + return - UUID of type 3 as a UUID record, or else a `uuid:Error`
public isolated function createType3AsRecord(NamespaceUUID namespace, string name) returns Uuid|Error {
    string|Error uuid3 = createType3AsString(namespace, name);
    if uuid3 is string {
        return check toRecord(uuid3);
    } else {
        return error Error("Failed to create UUID of type 3", uuid3);
    }
}

# Returns a UUID of type 4 as a string.
# ```ballerina
# string uuid4 = uuid:createType4AsString();
# ```
#
# + return - UUID of type 4 as a string
public isolated function createType4AsString() returns string {
    return externCreateType4AsString();
}

# Returns a UUID of type 4 as a UUID record.
# ```ballerina
# uuid:Uuid uuid4 = check uuid:createType4AsRecord();
# ```
#
# + return - UUID of type 4 as a UUID record, or else a `uuid:Error`
public isolated function createType4AsRecord() returns Uuid|Error {
    return check toRecord(createType4AsString());
}

# Returns a UUID of type 5 as a string.
# ```ballerina
# string uuid5 = check uuid:createType5AsString(uuid:NAME_SPACE_DNS, "ballerina.io");
# ```
#
# + namespace - String representation for a pre-defined namespace UUID
# + name - A name within the namespace
#
# + return - UUID of type 5 as a string, or else a `uuid:Error`
public isolated function createType5AsString(NamespaceUUID namespace, string name) returns string|Error {
    string trimmedName = name.trim();
    if trimmedName.length() == 0 {
        return error Error("Name cannot be empty");
    }
    byte[] namespaceBytes = check getBytesFromUuid(namespace);
    byte[] nameBytes = trimmedName.toBytes();
    foreach byte b in nameBytes {
        namespaceBytes.push(b);
    }

    byte[] uuid5 = crypto:hashSha1(namespaceBytes);

    uuid5[6] = uuid5[6] & 0x0f;
    uuid5[6] = <byte>(uuid5[6] | 0x50);
    uuid5[8] = uuid5[8] & 0x3f;
    uuid5[8] = <byte>(uuid5[8] | 0x80);
    return getUuidFromBytes(uuid5);
}

# Returns a UUID of type 5 as a UUID record.
# ```ballerina
# uuid:Uuid uuid5 = check uuid:createType5AsRecord(uuid:NAME_SPACE_DNS, "ballerina.io");
# ```
#
# + namespace - String representation for a pre-defined namespace UUID
# + name - A name within the namespace
#
# + return - UUID of type 5 as a UUID record, or else a `uuid:Error`
public isolated function createType5AsRecord(NamespaceUUID namespace, string name) returns Uuid|Error {
    string|Error uuid5 = createType5AsString(namespace, name);
    if uuid5 is string {
        return check toRecord(uuid5);
    } else {
        return error Error("Failed to create UUID of type 5", uuid5);
    }
}

# Returns a UUID of type 4 as a string.
# This function provides a convenient alias for `createType4AsString()`.
# ```ballerina
# string newUUID = uuid:createRandomUuid();
# ```
#
# + return - UUID of type 4 as a string
public isolated function createRandomUuid() returns string {
    return createType4AsString();
}

# Returns a nil UUID as a string.
# ```ballerina
# string nilUUID = uuid:nilAsString();
# ```
#
# + return - nil UUID as a string
public isolated function nilAsString() returns string {
    return NIL_UUID;
}

# Returns a nil UUID as a UUID record.
# ```ballerina
# uuid:Uuid nilUUID = uuid:nilAsRecord();
# ```
#
# + return - nil UUID as a UUID record
public isolated function nilAsRecord() returns Uuid {
    Uuid nilUuid = {
        timeLow: 0,
        timeMid: 0,
        timeHiAndVersion: 0,
        clockSeqHiAndReserved: 0,
        clockSeqLo: 0,
        node: 0
    };
    return nilUuid;
}

# Tests a string to see if it is a valid UUID.
# ```ballerina
# boolean valid = uuid:validate("4397465e-35f9-11eb-adc1-0242ac120002");
# ```
#
# + uuid - UUID string to be validated
#
# + return - true if a valid UUID, false if not
public isolated function validate(string uuid) returns boolean {
    return externValidate(uuid);
}

# Detects the RFC version of a UUID. Returns an error if the UUID is invalid.
# ```ballerina
# uuid:Version v = check uuid:getVersion("4397465e-35f9-11eb-adc1-0242ac120002");
# ```
#
# + uuid - UUID string to be checked
#
# + return - UUID version, or else a `uuid:Error`
public isolated function getVersion(string uuid) returns Version|Error {
    if !validate(uuid) {
        return error Error("Invalid UUID string provided");
    }

    Uuid u = check toRecord(uuid);

    int mostSigBits = u.timeLow & 0xffffffff;
    mostSigBits <<= 16;
    mostSigBits |= u.timeMid & 0xffff;
    mostSigBits <<= 16;
    mostSigBits |= u.timeHiAndVersion & 0xffff;

    int v = (mostSigBits >> 12) & 0x0f;

    match v {
        1 => {
            return V1;
        }
        3 => {
            return V3;
        }
        4 => {
            return V4;
        }
        5 => {
            return V5;
        }
        _ => {
            return error Error("Unsupported UUID version");
        }
    }
}

# Converts to an array of bytes. Returns an error if the UUID is invalid.
# ```ballerina
# byte[] b = check uuid:toBytes("4397465e-35f9-11eb-adc1-0242ac120002");
# ```
#
# + uuid - UUID to be converted
#
# + return - UUID as bytes, or else a `uuid:Error`
public isolated function toBytes(string|Uuid uuid) returns byte[]|Error {
    if uuid is string {
        if !validate(uuid) {
            return error Error("Invalid UUID string provided");
        }
        return getBytesFromUuid(uuid);
    } else {
        var uuidString = toString(uuid);
        if uuidString is string {
            return getBytesFromUuid(uuidString);
        } else {
            return error Error("Failed to convert UUID record to a string", uuidString);
        }
    }
}

# Converts to a UUID string. Returns an error if the UUID is invalid.
# ```ballerina
# byte[] b = check uuid:toBytes("550e8400-e29b-41d4-a716-446655440000");
# string s = check uuid:toString(b);
# ```
#
# + uuid - UUID to be converted
#
# + return - UUID as string, or else a `uuid:Error`
public isolated function toString(byte[]|Uuid uuid) returns string|error {
    if uuid is byte[] {
        if uuid.length() != 16 {
            return error Error("Invalid UUID byte array provided, expected 16 bytes");
        }
        return getUuidFromBytes(uuid);
    }
    Uuid u = uuid;
    return constructComponent(u.timeLow.toHexString(), 8) + "-" +
        constructComponent(u.timeMid.toHexString(), 4) + "-" +
        constructComponent(u.timeHiAndVersion.toHexString(), 4) + "-" +
        constructComponent(u.clockSeqHiAndReserved.toHexString(), 2) +
        constructComponent(u.clockSeqLo.toHexString(), 2) + "-" +
        constructComponent(u.node.toHexString(), 12);
}

# Converts to a UUID record. Returns an error if the UUID is invalid.
# ```ballerina
# uuid:Uuid r1 = check uuid:toRecord("4397465e-35f9-11eb-adc1-0242ac120002");
# byte[] b = check uuid:toBytes(r1);
# uuid:Uuid r2 = check uuid:toRecord(b);
# ```
#
# + uuid - UUID to be converted
#
# + return - UUID as record, or else a `uuid:Error`
public isolated function toRecord(string|byte[] uuid) returns Uuid|Error {
    string uuidStr;
    if uuid is string {
        if !validate(uuid) {
            return error Error("Invalid UUID string provided");
        }
        uuidStr = uuid;
    } else {
        if uuid.length() != 16 {
            return error Error("Invalid UUID byte array provided, expected 16 bytes");
        }
        uuidStr = getUuidFromBytes(uuid);
    }

    // UUID format: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
    string part0 = uuidStr.substring(0, 8);
    string part1 = uuidStr.substring(9, 13);
    string part2 = uuidStr.substring(14, 18);
    string part3 = uuidStr.substring(19, 23);
    string part4 = uuidStr.substring(24, 36);

    int|error timeLowIntFromHexString = externParseHexUint(part0);
    if timeLowIntFromHexString is error {
        return error Error("Failed to get int value of time-low hex string", timeLowIntFromHexString);
    }
    ints:Unsigned32 timeLowInt = <ints:Unsigned32>timeLowIntFromHexString;

    int|error timeMidIntFromHexString = externParseHexUint(part1);
    if timeMidIntFromHexString is error {
        return error Error("Failed to get int value of time-mid hex string", timeMidIntFromHexString);
    }
    ints:Unsigned16 timeMidInt = <ints:Unsigned16>timeMidIntFromHexString;

    int|error timeHiAndVersionIntFromHexString = externParseHexUint(part2);
    if timeHiAndVersionIntFromHexString is error {
        return error Error("Failed to get int value of time-hi-and-version hex string", timeHiAndVersionIntFromHexString);
    }
    ints:Unsigned16 timeHiAndVersionInt = <ints:Unsigned16>timeHiAndVersionIntFromHexString;

    int|error clockSeqHiAndReservedIntFromHexString = externParseHexUint(part3.substring(0, 2));
    if clockSeqHiAndReservedIntFromHexString is error {
        return error Error("Failed to get int value of clock-seq-hi-and-reserved hex string",
            clockSeqHiAndReservedIntFromHexString);
    }
    ints:Unsigned8 clockSeqHiAndReservedInt = <ints:Unsigned8>clockSeqHiAndReservedIntFromHexString;

    int|error clockSeqLoIntFromHexString = externParseHexUint(part3.substring(2, 4));
    if clockSeqLoIntFromHexString is error {
        return error Error("Failed to get int value of clock-seq-lo hex string", clockSeqLoIntFromHexString);
    }
    ints:Unsigned8 clockSeqLoInt = <ints:Unsigned8>clockSeqLoIntFromHexString;

    int|error nodeResult = externParseHexUint(part4);
    if nodeResult is error {
        return error Error("Failed to get int value of node string", nodeResult);
    }
    int nodeInt = nodeResult;

    Uuid uuidRecord = {
        timeLow: timeLowInt,
        timeMid: timeMidInt,
        timeHiAndVersion: timeHiAndVersionInt,
        clockSeqHiAndReserved: clockSeqHiAndReservedInt,
        clockSeqLo: clockSeqLoInt,
        node: nodeInt
    };
    return uuidRecord;
}

isolated function getBytesFromUuid(string uuid) returns byte[]|Error {
    Uuid uuidRecord = check toRecord(uuid);

    int msb = check getMostSigBits(uuidRecord);
    int lsb = check getLeastSigBits(uuid, uuidRecord);
    return bitsToBytes(msb, lsb);
}

isolated function getMostSigBits(Uuid uuid) returns int|Error {
    int mostSigBits = uuid.timeLow & 0xffffffff;
    mostSigBits <<= 16;
    mostSigBits |= uuid.timeMid & 0xffff;
    mostSigBits <<= 16;
    mostSigBits |= uuid.timeHiAndVersion & 0xffff;
    return mostSigBits;
}

isolated function getLeastSigBits(string uuidString, Uuid uuidRecord) returns int|Error {
    // UUID format: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
    // clock-seq field spans positions 19-22 (0-indexed, exclusive end)
    string clockSeq = uuidString.substring(19, 23);
    int|error clockSeqInt = externParseHexUint(clockSeq);
    int leastSigBits;
    if clockSeqInt is int {
        leastSigBits = clockSeqInt & 0xffff;
    } else {
        return error Error("Failed to get clock sequence value of the uuid");
    }

    leastSigBits <<= 48;
    leastSigBits |= uuidRecord.node & 0xffffffffffff;
    return leastSigBits;
}

isolated function getUuidFromBytes(byte[] uuid) returns string {
    int msb = ((uuid[0] & 0xFF) << 56) |
            ((uuid[1] & 0xFF) << 48) |
            ((uuid[2] & 0xFF) << 40) |
            ((uuid[3] & 0xFF) << 32) |
            ((uuid[4] & 0xFF) << 24) |
            ((uuid[5] & 0xFF) << 16) |
            ((uuid[6] & 0xFF) << 8) |
            ((uuid[7] & 0xFF) << 0);

    int lsb = ((uuid[8] & 0xFF) << 56) |
            ((uuid[9] & 0xFF) << 48) |
            ((uuid[10] & 0xFF) << 40) |
            ((uuid[11] & 0xFF) << 32) |
            ((uuid[12] & 0xFF) << 24) |
            ((uuid[13] & 0xFF) << 16) |
            ((uuid[14] & 0xFF) << 8) |
            ((uuid[15] & 0xFF) << 0);

    return bitsToUuid(msb, lsb);
}

isolated function bitsToBytes(int msb, int lsb) returns byte[] {
    byte[] result = [];

    result[0] = <byte>((msb >> 56) & 0xff);
    result[1] = <byte>((msb >> 48) & 0xff);
    result[2] = <byte>((msb >> 40) & 0xff);
    result[3] = <byte>((msb >> 32) & 0xff);
    result[4] = <byte>((msb >> 24) & 0xff);
    result[5] = <byte>((msb >> 16) & 0xff);
    result[6] = <byte>((msb >> 8) & 0xff);
    result[7] = <byte>((msb >> 0) & 0xff);

    result[8] = <byte>((lsb >> 56) & 0xff);
    result[9] = <byte>((lsb >> 48) & 0xff);
    result[10] = <byte>((lsb >> 40) & 0xff);
    result[11] = <byte>((lsb >> 32) & 0xff);
    result[12] = <byte>((lsb >> 24) & 0xff);
    result[13] = <byte>((lsb >> 16) & 0xff);
    result[14] = <byte>((lsb >> 8) & 0xff);
    result[15] = <byte>((lsb >> 0) & 0xff);

    return result;
}

// bitsToUuid converts MSB and LSB integers into a UUID string.
// Layout: (msb>>>32):8h-(msb>>>16)&ffff:4h-msb&ffff:4h-(lsb>>>48)&ffff:4h-lsb&ffffffffffff:12h
isolated function bitsToUuid(int mostSigBits, int leastSigBits) returns string {
    return constructComponent(((mostSigBits >>> 32) & 0xffffffff).toHexString(), 8) + "-" +
           constructComponent(((mostSigBits >>> 16) & 0xffff).toHexString(), 4) + "-" +
           constructComponent((mostSigBits & 0xffff).toHexString(), 4) + "-" +
           constructComponent(((leastSigBits >>> 48) & 0xffff).toHexString(), 4) + "-" +
           constructComponent((leastSigBits & 0xffffffffffff).toHexString(), 12);
}

isolated function constructComponent(string hex, int length) returns string {
    string hexString = "";
    foreach int _ in 0 ..< (length - hex.length()) {
        hexString += "0";
    }
    return hexString + hex;
}

isolated function externCreateType1AsString() returns string = external;

isolated function externCreateType4AsString() returns string = external;

isolated function externValidate(string uuid) returns boolean = external;

isolated function externParseHexUint(string hex) returns int|error = external;
