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
import ballerina/mime;

public function main() {
    mime:ContentDisposition cd = mime:getContentDispositionObject("form-data; name=\"file\"; filename=\"test.txt\"");
    io:println(cd.disposition);
    io:println(cd.name);
    io:println(cd.fileName);

    mime:ContentDisposition newCd = new ();
    newCd.disposition = "attachment";
    newCd.fileName = "report.pdf";
    io:println(newCd.toString());

    mime:ContentDisposition quotedCd = new ();
    quotedCd.disposition = "attachment";
    quotedCd.fileName = "my report.pdf";
    io:println(quotedCd.toString());
}
// @output form-data
// @output file
// @output test.txt
// @output attachment; filename=report.pdf
// @output attachment; filename="my report.pdf"
