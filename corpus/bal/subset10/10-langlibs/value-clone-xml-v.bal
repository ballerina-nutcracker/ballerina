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
    xml element = xml `<p xmlns:ns="urn:x" id="1">text<!--c--></p>`;
    xml elementClone = element.clone();
    io:println(elementClone); // @output <p id="1" xmlns:ns="urn:x">text<!--c--></p>
    io:println(elementClone == element); // @output true
    io:println(elementClone === element); // @output false

    xml first = xml `<a/>`;
    xml second = xml `<b/>`;
    xml sequence = first + second;
    xml sequenceClone = sequence.clone();
    io:println(sequenceClone); // @output <a/><b/>
    io:println(sequenceClone == sequence); // @output true
    io:println(sequenceClone === sequence); // @output false
    // The clone's items are new values, not the sources.
    foreach xml item in sequenceClone {
        io:println(item === first); // @output false
                                    // @output false
        io:println(item === second); // @output false
                                     // @output false
    }

    xml comment = xml `<!--note-->`;
    io:println(comment.clone() == comment); // @output true
    io:println(comment.clone() === comment); // @output false

    xml instruction = xml `<?target data?>`;
    io:println(instruction.clone() == instruction); // @output true
    io:println(instruction.clone() === instruction); // @output false

    // xml text is inherently immutable, so its clone is the same value.
    xml text = xml `hello`;
    io:println(text.clone() === text); // @output true
}
