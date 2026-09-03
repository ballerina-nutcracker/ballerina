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
    xml root = xml `<root><a-b>hyphen</a-b><u005C>weird</u005C></root>`;
    xml children = root/*;

    io:println(root/<a\-b>); // @output <a-b>hyphen</a-b>
    io:println(root/<a\u{2D}b>); // @output <a-b>hyphen</a-b>
    io:println(root/<u005C>); // @output <u005C>weird</u005C>

    // `\u{005C}` is a backslash, so it must not match an element named `u005C`.
    io:println(root/<\u{005C}>); // @output
    io:println(children.<\u{005C}>); // @output
    io:println(children.<u005C|\u{005C}>); // @output <u005C>weird</u005C>
}
