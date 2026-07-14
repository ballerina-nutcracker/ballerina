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
    xml:Element e1 = xml `<e>tt</e>`;
    xml:Element e2 = xml `<e2><ee>tt</ee><xx>xx</xx></e2>`;
    xml<xml:Element> e3 = e1 + e2;
    xml:ProcessingInstruction pe = xml `<?pe?>`;
    xml<xml:Element|xml:ProcessingInstruction> e4 = e3 + pe;
    io:println(e4); // @output <e>tt</e><e2><ee>tt</ee><xx>xx</xx></e2><?pe ?>
}
