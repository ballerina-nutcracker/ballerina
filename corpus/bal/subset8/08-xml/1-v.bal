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
    xml:ProcessingInstruction pe = xml `<?pe?>`;
    xml:Element e1 = xml `<element>test</element>`;
    xml e2 = xml `<other/>`;
    xml text = xml `bar`;
    xml sequence = pe + e1 + e2 + text;
    io:println(sequence); // @output <?pe ?><element>test</element><other/>bar
}

