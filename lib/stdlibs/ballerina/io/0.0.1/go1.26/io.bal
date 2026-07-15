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

# Represents the standard output stream.
public const stdout = 1;

# Represents the standard error stream.
public const stderr = 2;

// Defines the output streaming types.
// 1. `stdout` - standard output stream
// 2. `stderr` - standard error stream
public type FileOutputStream stdout|stderr;

// Defines all the printable types.
// 1. any typed value
// 2. errors
public type Printable any|error;

// Represents io module related errors.
// Note: distinct error subtypes are not yet supported; Error is currently an alias for error.
public type Error error;

// Represents file write options.
// OVERWRITE truncates and overwrites the file; APPEND adds to the existing content.
public enum FileWriteOption {
    OVERWRITE,
    APPEND
}

# Prints `any` or `error` to the standard output stream.
# ```ballerina
# io:print("Start processing the CSV file from ", srcFileName);
# ```
#
# + values - The value(s) to be printed
public isolated function print(Printable... values) {
    externPrint(stdout, false, values);
}

# Prints `any` or `error` to the standard output stream and terminates the line.
# ```ballerina
# io:println("Start processing the CSV file from ", srcFileName);
# ```
#
# + values - The value(s) to be printed
public isolated function println(Printable... values) {
    externPrint(stdout, true, values);
}

# Prints `any`, `error`, or string templates value(s) to a given stream(STDOUT or STDERR).
# ```ballerina
# io:fprint(io:stderr, "Unexpected error occurred");
# ```
# + fileOutputStream - The output stream (`io:stdout` or `io:stderr`) content needs to be printed
# + values - The value(s) to be printed
public isolated function fprint(FileOutputStream fileOutputStream, Printable... values) {
    externPrint(fileOutputStream, false, values);
}

# Prints `any`, `error`, or string templates value(s) to a given stream(STDOUT or STDERR) and terminates the line.
# ```ballerina
# io:fprintln(io:stderr, "Unexpected error occurred");
# ```
# + fileOutputStream - The output stream (`io:stdout` or `io:stderr`) content needs to be printed
# + values - The value(s) to be printed
public isolated function fprintln(FileOutputStream fileOutputStream, Printable... values) {
    externPrint(fileOutputStream, true, values);
}

isolated function externPrint(FileOutputStream fileOutputStream, boolean newLine, Printable... values) = external;

# Reads the entire file content as a `string`.
# The resulting string does not contain terminal carriage characters (`\r` or `\r\n`); line endings are normalised to `\n`.
# ```ballerina
# string|io:Error content = io:fileReadString("./resources/myfile.txt");
# ```
# + path - The file path
# + return - The entire file content as a string or an `io:Error`
public isolated function fileReadString(string path) returns string|Error {
    return externFileReadString(path);
}

# Reads the entire file content as a list of lines.
# The resulting array does not contain terminal carriage characters (`\r` or `\r\n`).
# ```ballerina
# string[]|io:Error lines = io:fileReadLines("./resources/myfile.txt");
# ```
# + path - The file path
# + return - The file content as a string array or an `io:Error`
public isolated function fileReadLines(string path) returns string[]|Error {
    return externFileReadLines(path);
}

# Reads the entire file content as a byte array.
# ```ballerina
# byte[]|io:Error content = io:fileReadBytes("./resources/myfile.txt");
# ```
# + path - The file path
# + return - A byte array or an `io:Error`
public isolated function fileReadBytes(string path) returns byte[]|Error {
    return externFileReadBytes(path);
}

# Reads file content as a JSON.
# ```ballerina
# json|io:Error content = io:fileReadJson("./resources/myfile.json");
# ```
# + path - The JSON file path
# + return - The file content as a JSON or an `io:Error`
public isolated function fileReadJson(string path) returns json|Error {
    return externFileReadJson(path);
}

# Writes a string to a file.
# ```ballerina
# io:Error? result = io:fileWriteString("./resources/myfile.txt", "Hello World");
# ```
# + path - The file path
# + content - String content to write
# + option - Whether to overwrite or append the given content (default: `OVERWRITE`)
# + return - `()` when the write was successful or an `io:Error`
public isolated function fileWriteString(string path, string content, FileWriteOption option = OVERWRITE) returns Error? {
    return externFileWriteString(path, content, option);
}

# Writes an array of lines to a file.
# A newline character `\n` is appended after each line.
# ```ballerina
# io:Error? result = io:fileWriteLines("./resources/myfile.txt", ["Hello", "World"]);
# ```
# + path - The file path
# + content - An array of string lines to write
# + option - Whether to overwrite or append the given content (default: `OVERWRITE`)
# + return - `()` when the write was successful or an `io:Error`
public isolated function fileWriteLines(string path, string[] content, FileWriteOption option = OVERWRITE) returns Error? {
    return externFileWriteLines(path, content, option);
}

# Writes a byte array to a file.
# ```ballerina
# io:Error? result = io:fileWriteBytes("./resources/myfile.bin", [72, 101, 108, 108, 111]);
# ```
# + path - The file path
# + content - Byte content to write
# + option - Whether to overwrite or append the given content (default: `OVERWRITE`)
# + return - `()` when the write was successful or an `io:Error`
public isolated function fileWriteBytes(string path, byte[] content, FileWriteOption option = OVERWRITE) returns Error? {
    return externFileWriteBytes(path, content, option);
}

# Writes a JSON to a file.
# ```ballerina
# io:Error? result = io:fileWriteJson("./resources/myfile.json", {"name": "Alice"});
# ```
# + path - The JSON file path
# + content - JSON content to write
# + return - `()` when the write was successful or an `io:Error`
public isolated function fileWriteJson(string path, json content) returns Error? {
    return externFileWriteJson(path, content);
}

# Reads file content as an `xml`.
# ```ballerina
# xml|io:Error content = io:fileReadXml("./resources/myfile.xml");
# ```
# + path - The XML file path
# + return - The file content as an `xml` or an `io:Error`
public isolated function fileReadXml(string path) returns xml|Error {
    return externFileReadXml(path);
}

# Writes an `xml` to a file.
# ```ballerina
# io:Error? result = io:fileWriteXml("./resources/myfile.xml", xml `<book><title>Clean Code</title></book>`);
# ```
# + path - The XML file path
# + content - XML content to write
# + fileWriteOption - Whether to overwrite or append the given content (default: `OVERWRITE`)
# + return - `()` when the write was successful or an `io:Error`
public isolated function fileWriteXml(string path, xml content, FileWriteOption fileWriteOption = OVERWRITE) returns Error? {
    return externFileWriteXml(path, content, fileWriteOption);
}

isolated function externFileReadString(string path) returns string|Error = external;
isolated function externFileReadLines(string path) returns string[]|Error = external;
isolated function externFileReadBytes(string path) returns byte[]|Error = external;
isolated function externFileReadJson(string path) returns json|Error = external;
isolated function externFileWriteString(string path, string content, FileWriteOption option) returns Error? = external;
isolated function externFileWriteLines(string path, string[] content, FileWriteOption option) returns Error? = external;
isolated function externFileWriteBytes(string path, byte[] content, FileWriteOption option) returns Error? = external;
isolated function externFileWriteJson(string path, json content) returns Error? = external;
isolated function externFileReadXml(string path) returns xml|Error = external;
isolated function externFileWriteXml(string path, xml content, FileWriteOption option) returns Error? = external;
