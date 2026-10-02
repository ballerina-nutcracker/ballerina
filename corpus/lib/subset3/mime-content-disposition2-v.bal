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
    any parsed = mime:getContentDispositionObject("attachment");
    io:println(parsed is mime:ContentDisposition); // @output true

    // The disposition keeps its case; only name and filename are unquoted.
    mime:ContentDisposition cd = mime:getContentDispositionObject("Form-Data; name=\"a b\"; filename=f.txt; size=10; X=\"y\"");
    io:println(cd.disposition); // @output Form-Data
    io:println(cd.name); // @output a b
    io:println(cd.fileName); // @output f.txt
    io:println(cd.parameters["size"], " ", cd.parameters["X"]); // @output 10 "y"
    io:println(cd.toString()); // @output Form-Data;name="a b";filename="f.txt";size=10;X="y"

    // name and filename are quoted unless already quoted; other parameters are written as-is.
    mime:ContentDisposition built = new;
    built.disposition = "attachment";
    built.name = "\"q\"";
    built.parameters = {"k": "v w", "z": "1"};
    io:println(built.toString()); // @output attachment;name="q";k=v w;z=1

    // An empty disposition serializes to the empty string.
    mime:ContentDisposition empty = mime:getContentDispositionObject("");
    io:println(empty.toString().length()); // @output 0

    mime:Entity entity = new;
    mime:ContentDisposition upload = new;
    upload.disposition = "form-data";
    upload.name = "upload";
    upload.fileName = "file.txt";
    entity.setContentDisposition(upload);
    io:println(entity.getHeader("content-disposition")); // @output form-data;name="upload";filename="file.txt"
}
