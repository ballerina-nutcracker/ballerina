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
import ballerina/url;

public function main() returns error? {
    // Standard percent-encoded query string
    string decoded = check url:decode("param1%3Dhttp%3A%2F%2Fxyz.com%2F%3Fa%3D12%26b%3D55", "UTF-8");
    io:println(decoded); // @output param1=http://xyz.com/?a=12&b=55

    // %20 → space
    io:println(check url:decode("hello%20world", "UTF-8")); // @output hello world

    // + → space (URLDecoder compatibility)
    io:println(check url:decode("hello+world", "UTF-8")); // @output hello world

    // %2A → *
    io:println(check url:decode("a%2Ab", "UTF-8")); // @output a*b

    // %7E → ~ (even though encode never produces %7E, decode still handles it)
    io:println(check url:decode("a%7Eb", "UTF-8")); // @output a~b
}
