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

const string SCHEME = "https:";
const string URI = SCHEME + "foo/" + "bar";

xmlns URI as bar;

xml elem = xml `<bar:f><bar:x></bar:x></bar:f>`;

public function main() {
    io:println(elem); // @output <bar:f xmlns:bar="https:foo/bar"><bar:x/></bar:f>

    xmlns URI as local;
    xml localElem = xml `<local:f/>`;
    io:println(localElem); // @output <local:f xmlns:local="https:foo/bar"/>
}
