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

// `endIndex` of xml:slice defaults to the length of `x`, so omitting it slices
// to the end of the sequence. The container argument is evaluated once even
// though the default is computed from it.
int counter = 0;

function items() returns xml<xml:Element> {
    counter += 1;
    io:println("eval");
    return xml `<a/><b/><c/>`;
}

public function main() {
    xml x = xml `<a/><b/><c/>`;
    io:println(x.slice(1)); // @output <b/><c/>
    io:println(xml:slice(x, 2)); // @output <c/>
    io:println(xml:slice(x, startIndex = 1)); // @output <b/><c/>
    io:println(x.slice(3).length()); // @output 0
    xml<xml:Comment> comments = xml `<!--p--><!--q-->`;
    xml<xml:Comment> lastComment = comments.slice(1);
    io:println(lastComment); // @output <!--q-->
    xml empty = xml ``;
    io:println(empty.slice(0).length()); // @output 0
    xml r = xml `<r><p/><q/></r>`;
    io:println(r/*.slice(1)); // @output <q/>

    // @output eval
    xml<xml:Element> tail = items().slice(1);
    io:println(tail); // @output <b/><c/>
    io:println(counter); // @output 1
}
