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

import ballerina/io;

public function main() {
    xml? value = ();
    string text = "text";
    xml? result = value + text;
    io:println(result is ()); // @output true

    value = xml `<a/>`;
    io:println(value + text); // @output <a/>text

    string? prefix = ();
    result = prefix + xml `x`;
    io:println(result is ()); // @output true

    prefix = "p";
    io:println(prefix + xml `x`); // @output px
}
