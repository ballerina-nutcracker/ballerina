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
    string path = "/tmp/bal_io_xml2.xml";
    xml data = xml `<root attr="val"><child/></root>`;
    check io:fileWriteXml(path, data);
    xml result = check io:fileReadXml(path);
    io:println(result);
    // Overwrite with different content
    xml data2 = xml `<updated/>`;
    check io:fileWriteXml(path, data2);
    xml result2 = check io:fileReadXml(path);
    io:println(result2);
}
// @output <root attr="val"><child/></root>
// @output <updated/>
