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

import ballerina/io;

public function main() returns error? {
    // A float JSON scalar round-trips: write goes through balValueToGoJSON float64,
    // read parses a non-integer json.Number back to a float.
    string floatPath = "/tmp/bal_io_json_float.json";
    check io:fileWriteJson(floatPath, 3.14);
    json floatBack = check io:fileReadJson(floatPath);
    io:println(floatBack); // @output 3.14

    // A decimal JSON scalar round-trips: write emits the decimal verbatim.
    string decimalPath = "/tmp/bal_io_json_decimal.json";
    check io:fileWriteJson(decimalPath, 2.5d);
    json decimalBack = check io:fileReadJson(decimalPath);
    io:println(decimalBack); // @output 2.5

    // An integer scalar reads back as an int (json.Number Int64 path).
    string intPath = "/tmp/bal_io_json_int.json";
    check io:fileWriteJson(intPath, 42);
    json intBack = check io:fileReadJson(intPath);
    io:println(intBack); // @output 42

    // A JSON array with mixed numeric/boolean/null/string members -> list conversion
    // on both write (balValueToGoJSON list) and read (goJSONToBalValue []any).
    string arrPath = "/tmp/bal_io_json_arr.json";
    json arr = [1, 2.5, true, (), "x"];
    check io:fileWriteJson(arrPath, arr);
    json arrBack = check io:fileReadJson(arrPath);
    io:println(arrBack); // @output [1,2.5,true,null,"x"]
}
