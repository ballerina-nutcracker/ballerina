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
    xml<xml:Element|xml:Text|xml:Comment> x = xml `<a/>hello<!--c-->`;

    record {| xml:Element|xml:Text|xml:Comment value; |}? methodNext = x.iterator().next();
    if methodNext != () {
        io:println(methodNext.value); // @output <a/>
    }

    record {| xml:Element|xml:Text|xml:Comment value; |}? functionNext = xml:iterator(x).next();
    if functionNext != () {
        io:println(functionNext.value); // @output <a/>
    }

    foreach xml:Element|xml:Text|xml:Comment item in x {
        io:println(item); // @output <a/>
                          // @output hello
                          // @output <!--c-->
    }

    xml:Element element = xml `<e/>`;
    record {| xml:Element value; |}? elementNext = element.iterator().next();
    if elementNext != () {
        io:println(elementNext.value); // @output <e/>
    }
    foreach xml:Element item in element {
        io:println(item); // @output <e/>
    }

    xml:Comment comment = xml `<!--comment-->`;
    record {| xml:Comment value; |}? commentNext = comment.iterator().next();
    if commentNext != () {
        io:println(commentNext.value); // @output <!--comment-->
    }
    foreach xml:Comment item in comment {
        io:println(item); // @output <!--comment-->
    }

    xml:ProcessingInstruction pi = xml `<?p data?>`;
    record {| xml:ProcessingInstruction value; |}? piNext = pi.iterator().next();
    if piNext != () {
        io:println(piNext.value); // @output <?p data?>
    }
    foreach xml:ProcessingInstruction item in pi {
        io:println(item); // @output <?p data?>
    }

    xml:Text text = xml `text`;
    record {| xml:Text value; |}? textNext = text.iterator().next();
    if textNext != () {
        io:println(textNext.value); // @output text
    }
    foreach xml:Text item in text {
        io:println(item); // @output text
    }
}
