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

# Parse a CSV string as a subtype of `record {}[]` or `anydata[][]`.
# 
# ```ballerina
# string csvString = string `id,name
#                            1,John
#                            3,Jane`;
# record {int id; string name;}[] csv1 = check csv:parseString(csvString);
# [int, string][] csv2 = check csv:parseString(csvString);
# record {|int id;|}[] csv3 = check csv:parseString(csvString);
# record {int id;}[] csv4 = check csv:parseString(csvString, {skipLines: [1]});
# ```
#
# + csvString - Source CSV string value
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, value belonging to the given target type, else returns an `csv:Error` value.
public isolated function parseString(string csvString, ParseOptions options = {}, typedesc<record {}[]|anydata[][]> t = <>)
     returns t|Error = external;

# Parse a byte[] as a subtype of `record {}[]` or `anydata[][]`.
# 
# ```ballerina
# byte[] csvBytes = check io:fileReadBytes("example.csv");
# 
# record {int id; string name;}[] csv1 = check csv:parseBytes(csvBytes);
# [int, string][] csv2 = check csv:parseBytes(csvBytes);
# record {|int id;|}[] csv3 = check csv:parseBytes(csvBytes);
# record {int id;}[] csv4 = check csv:parseBytes(csvBytes, {skipLines: [1]});
# ```
#
# + csvBytes - Source CSV byte array
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, value belonging to the given target type, else returns an `csv:Error` value.
public isolated function parseBytes(byte[] csvBytes, ParseOptions options = {}, typedesc<record {}[]|anydata[][]> t = <>)
     returns t|Error = external;

# Parse a CSV byte block stream as a subtype of `record {}[]` or `anydata[][]`.
# 
# ```ballerina
# stream<byte[], io:Error?> csvByteStream = check io:fileReadBlocksAsStream("example.csv");
# record {int id; string name;}[] csv1 = check csv:parseStream(csvByteStream);
# 
# stream<byte[], io:Error?> csvByteStream2 = check io:fileReadBlocksAsStream("example.csv");
# [int, string][] csv2 = check csv:parseStream(csvByteStream2);
# 
# stream<byte[], io:Error?> csvByteStream3 = check io:fileReadBlocksAsStream("example.csv");
# record {|int id;|}[] csv3 = check csv:parseStream(csvByteStream3);
# 
# stream<byte[], io:Error?> csvByteStream4 = check io:fileReadBlocksAsStream("example.csv");
# record {int id;}[] csv4 = check csv:parseStream(csvByteStream4, {skipLines: [1]});
# ```
# 
# + csvByteStream - Source CSV byte block stream
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, value belonging to the given target type, else returns an `csv:Error` value.
public isolated function parseStream(stream<byte[], error?> csvByteStream,
           ParseOptions options = {}, typedesc<record {}[]|anydata[][]> t = <>)
     returns t|Error = external;

# Transform value of type record {}[] to subtype of `record {}[]` or `anydata[][]`.
# 
# ```ballerina
# record {int id; string name;}[] csvRecords = [{id: 1, name: "John"}, {id: 2, name: "Jane"}];
# [int, string][] csv1 = check csv:transform(csvRecords);
# record {|int id;|}[] csv2 = check csv:transform(csvRecords);
# record {int id;}[] csv3 = check csv:transform(csvRecords, {skipLines: [1]});
# ```
#
# + csvRecords - Source Ballerina record array value
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, returns value belonging to the given target type, else returns an `csv:Error` value.
public isolated function transform(record {}[] csvRecords,
           TransformOptions options = {}, typedesc<record {}[]|anydata[][]> t = <>)
     returns t|Error = external;

# Parse a string array of array as a subtype of `record {}[]` or `anydata[][]`.
# 
# ```ballerina
# string[][] csvList = [["1", "John"], ["2", "Jane"]];
# [int, string][] csv1 = check csv:parseList(csvList);
# record {|int id;|}[] csv2 = check csv:parseList(csvList, {customHeaders: ["id", "name"]});
# record {int id;}[] csv3 = check csv:parseList(csvList, {skipLines: [1], customHeaders: ["id", "name"]});
# ```
# 
# + csvList - Source Ballerina string array of array value
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, returns value belonging to the given target type, else returns an `csv:Error` value.
public isolated function parseList(string[][] csvList, ParseListOptions options = {}, typedesc<record {}[]|anydata[][]> t = <>)
     returns t|Error = external;
