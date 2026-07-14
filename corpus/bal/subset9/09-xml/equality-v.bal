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
    xml e1 = xml `<a x="1" xmlns:p="urn:p"><p:b>text</p:b></a>`;
    xml e2 = xml `<a x="1" xmlns:p="urn:p"><p:b>text</p:b></a>`;
    xml e3 = e1;
    xml e4 = xml `<a x="2" xmlns:p="urn:p"><p:b>text</p:b></a>`;

    io:println(e1 == e2); // @output true
    io:println(e1 == e4); // @output false
    io:println(e1 === e3); // @output true
    io:println(e1 === e2); // @output false

    xml c1 = xml `<!--x-->`;
    xml c2 = xml `<!--x-->`;
    xml c3 = c1;
    io:println(c1 == c2); // @output true
    io:println(c1 === c3); // @output true
    io:println(c1 === c2); // @output false

    xml p1 = xml `<?p data?>`;
    xml p2 = xml `<?p data?>`;
    xml p3 = p1;
    io:println(p1 == p2); // @output true
    io:println(p1 === p3); // @output true
    io:println(p1 === p2); // @output false

    xml t1 = xml `hello`;
    xml t2 = xml `hello`;
    xml t3 = xml `world`;
    io:println(t1 == t2); // @output true
    io:println(t1 === t2); // @output true
    io:println(t1 == t3); // @output false

    xml s1 = xml `<a/>hello`;
    xml s2 = xml `<a/>hello`;
    xml s3 = xml `<a/>world`;
    io:println(s1 == s2); // @output true
    io:println(s1 === s2); // @output false
    io:println(s1 == s3); // @output false
}
