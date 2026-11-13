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
    string first = "first";
    string second = "second";
    string third = "third";
    record {| int spread; |} seed = {spread: 2};
    map<int> ordered = {[first]: 3, specific: 1, [second]: 4,
        ...seed, [first]: 5, [third]: 6};
    foreach string key in ordered.keys() {
        io:println(key, ":", ordered[key]);
    }
    // @output specific:1
    // @output spread:2
    // @output first:5
    // @output second:4
    // @output third:6
}
