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
    (xml & readonly)|map<int> element = xml `<a/>`;
    io:println(element); // @output <a/>
    io:println(isReadonly(element)); // @output true

    (xml & readonly)|map<int> sequence = xml `<a/><!--c-->`;
    io:println(sequence); // @output <a/><!--c-->
    io:println(isReadonly(sequence)); // @output true

    (xml & readonly)|map<int> interpolated = xml `<a>${"text"}</a>`;
    io:println(interpolated); // @output <a>text</a>
    io:println(isReadonly(interpolated)); // @output true

    xml|map<int> mutable = xml `<a/><!--c-->`;
    io:println(isReadonly(mutable)); // @output false
}

function isReadonly(any v) returns boolean {
    return v is readonly;
}
