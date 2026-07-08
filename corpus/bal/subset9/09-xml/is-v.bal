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
    xml elem = xml `<a/>`;
    xml comment = xml `<!--x-->`;
    xml pi = xml `<?foo bar?>`;
    xml text = xml `hello`;
    xml seq = xml `<a/>hello`;

    io:println(elem is xml:Element); // @output true
    io:println(comment is xml:Comment); // @output true
    io:println(pi is xml:ProcessingInstruction); // @output true
    io:println(text is xml:Text); // @output true
    io:println(seq is xml<xml:Element|xml:Text>); // @output true
}
