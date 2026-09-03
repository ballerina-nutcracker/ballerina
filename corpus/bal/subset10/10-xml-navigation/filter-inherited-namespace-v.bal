// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

public function main() {
    xmlns "urn:items" as p;
    xmlns "urn:items" as q;
    xml root = xml `<root xmlns:p="urn:items"><p:item/></root>`;
    xml children = root/*;
    io:println(children.<q:item>); // @output <item xmlns="urn:items"/>
}
