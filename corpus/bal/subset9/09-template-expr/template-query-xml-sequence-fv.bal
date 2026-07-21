// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import ballerina/io;

public function main() {
    int[] values = [1, 2];
    // TODO: Support XML sequence insertion from query result: https://github.com/ballerina-nutcracker/ballerina/issues/533
    xml x = xml `<root>${from var n in values select xml `<n>${n}</n>`}</root>`; // @error future: XML sequence insertion from query result
    io:println(x); // @output <root><n>1</n><n>2</n></root>
}
