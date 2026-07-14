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
    xml e1 = xml `<e a="x&amp;y&quot;z">hello&amp;world</e>`;
    io:println(e1); // @output <e a="x&amp;amp;y&amp;quot;z">hello&amp;amp;world</e>
    xml t = xml `pure text &amp; entity`;
    io:println(t); // @output pure text &amp;amp; entity
    xml comment = xml `<!-- safe comment -->`;
    io:println(comment); // @output <!-- safe comment -->
    xml pi = xml `<?target some data?>`;
    io:println(pi); // @output <?target some data?>
}
