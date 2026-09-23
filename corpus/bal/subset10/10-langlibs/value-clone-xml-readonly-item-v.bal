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

function same(any a, any b) returns boolean => a === b;

function firstItem(xml sequence) returns xml {
    xml[] items = [];
    foreach xml item in sequence {
        items.push(item);
    }
    return items[0];
}

public function main() {
    readonly & xml frozen = xml `<a/>`;
    xml sequence = frozen + xml `<b/>`;
    io:println(same(firstItem(sequence), frozen)); // @output true
    xml sequenceClone = sequence.clone();
    io:println(same(sequenceClone, sequence)); // @output false
    io:println(same(firstItem(sequenceClone), frozen)); // @output true
    io:println(sequenceClone); // @output <a/><b/>
}
