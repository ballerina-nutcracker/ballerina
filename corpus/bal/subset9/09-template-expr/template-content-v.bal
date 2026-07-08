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

xml XML_VALUE = xml `<root>${"x"}</root>`;

public function main() {
    string name = "world";
    xml x = xml `<greeting>Hello, ${name}!</greeting>`;
    io:println(x); // @output <greeting>Hello, world!</greeting>

    string markup = "<child/>";
    string amp = "a&b";
    xml escaped = xml `<root>${markup}|${amp}</root>`;
    io:println(escaped); // @output <root>&lt;child/&gt;|a&amp;b</root>
    io:println(XML_VALUE); // @output <root>x</root>
}
