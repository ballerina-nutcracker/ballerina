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

// A prefix defined by an inline xmlns on a non-root ancestor element should
// only be emitted on that ancestor, not hoisted onto the root element.
public function main() {
    xml elem = xml `<a><b xmlns:p="https:foo/u2"><p:c></p:c></b></a>`;
    io:println(elem); // @output <a><b xmlns:p="https:foo/u2"><p:c/></b></a>
}
