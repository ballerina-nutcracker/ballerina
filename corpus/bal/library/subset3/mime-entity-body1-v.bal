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
    mime:Entity textEntity = new ();
    textEntity.setText("Hello World");
    string|mime:ParserError textResult = textEntity.getText();
    if textResult is string {
        io:println(textResult);
    }

    mime:Entity bytesEntity = new ();
    bytesEntity.setByteArray([72, 101, 108, 108, 111]);
    byte[]|mime:ParserError bytesResult = bytesEntity.getByteArray();
    if bytesResult is byte[] {
        io:println(bytesResult.length());
    }

    mime:Entity bodyEntity = new ();
    bodyEntity.setBody("dispatched text");
    string|mime:ParserError dispatchResult = bodyEntity.getText();
    if dispatchResult is string {
        io:println(dispatchResult);
    }

    // Every accessor lazily converts from whatever the body was actually set as
    // (matching jBallerina's data-source model), so getByteArray() on a text-body
    // entity succeeds rather than erroring.
    mime:Entity crossKindEntity = new ();
    crossKindEntity.setText("text");
    byte[]|mime:ParserError crossKindResult = crossKindEntity.getByteArray();
    if crossKindResult is byte[] {
        io:println(crossKindResult.length());
    }

    // A freshly-constructed entity with no body set at all is the one case that
    // still produces a ParserError.
    mime:Entity emptyEntity = new ();
    byte[]|mime:ParserError emptyResult = emptyEntity.getByteArray();
    if emptyResult is mime:ParserError {
        io:println("parser error");
    }
}
// @output Hello World
// @output 5
// @output dispatched text
// @output 4
// @output parser error
