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

    // getJson() decodes only the first JSON value; trailing non-whitespace data is
    // rejected rather than silently dropped.
    mime:Entity trailingJsonEntity = new ();
    trailingJsonEntity.setByteArray("{} false".toBytes(), "application/json");
    json|mime:ParserError trailingJsonResult = trailingJsonEntity.getJson();
    if trailingJsonResult is mime:ParserError {
        io:println("trailing data rejected");
    }

    // Trailing whitespace after the JSON value is still accepted.
    mime:Entity trailingWsEntity = new ();
    trailingWsEntity.setByteArray("{}  \n".toBytes(), "application/json");
    json|mime:ParserError trailingWsResult = trailingWsEntity.getJson();
    if trailingWsResult is json {
        io:println("trailing whitespace accepted");
    }

    // setFileAsEntityBody sets the file as a lazy data source; default content-type
    // is application/octet-stream, overridable like the other setters.
    string filePath = "/tmp/bal_mime_entity_body_check.txt";
    checkpanic io:fileWriteString(filePath, "content from a file");
    mime:Entity fileEntity = new ();
    fileEntity.setFileAsEntityBody(filePath);
    io:println(fileEntity.getContentType());
    string|mime:ParserError fileTextResult = fileEntity.getText();
    if fileTextResult is string {
        io:println(fileTextResult);
    }

    mime:Entity fileEntityWithType = new ();
    fileEntityWithType.setFileAsEntityBody(filePath, "text/custom");
    io:println(fileEntityWithType.getContentType());

    // getByteArray() materializes a file-backed (channel) body too, not just getText().
    byte[]|mime:ParserError fileBytesResult = fileEntityWithType.getByteArray();
    if fileBytesResult is byte[] {
        io:println(fileBytesResult.length());
    }

    // The file's content is read on demand, not at setFileAsEntityBody time: a
    // change to the file between set and read is visible to the accessor,
    // matching jBallerina's lazy byte-channel data source.
    mime:Entity lazyEntity = new ();
    lazyEntity.setFileAsEntityBody(filePath);
    checkpanic io:fileWriteString(filePath, "overwritten content");
    string|mime:ParserError lazyResult = lazyEntity.getText();
    if lazyResult is string {
        io:println(lazyResult);
    }

    // The underlying byte channel can only be drained once, so the first accessor to
    // materialize it must cache the result on the entity (matching jBallerina's
    // EntityBodyHandler.updateDataSource) rather than re-reading — an already-exhausted
    // channel on the second read would otherwise silently yield an empty body.
    string|mime:ParserError cachedResult = lazyEntity.getText();
    io:println(cachedResult);
    byte[]|mime:ParserError cachedBytesResult = lazyEntity.getByteArray();
    if cachedBytesResult is byte[] {
        io:println(cachedBytesResult.length());
    }
}
// @output Hello World
// @output 5
// @output dispatched text
// @output 4
// @output parser error
// @output trailing data rejected
// @output trailing whitespace accepted
// @output application/octet-stream
// @output content from a file
// @output text/custom
// @output 19
// @output overwritten content
// @output overwritten content
// @output 19
