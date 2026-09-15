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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

public function main() {
    xmlns "foo" as ns;
    xmlns "bar" as k;

    xml single = xml `<ns:root></ns:root>`;
    io:println(single.<*>); // @output <ns:root xmlns:ns="foo"/>
    io:println(single.<ns:*>); // @output <ns:root xmlns:ns="foo"/>
    io:println(single.<ns:root>); // @output <ns:root xmlns:ns="foo"/>
    io:println(single.<ns:other>); // @output
    io:println(single.<other>); // @output
    io:println(single.<k:*>); // @output

    xml sequence = xml `<ns:root></ns:root><k:root></k:root><k:item></k:item>`;
    io:println(sequence.<*>); // @output <ns:root xmlns:ns="foo"/><k:root xmlns:k="bar"/><k:item xmlns:k="bar"/>
    io:println(sequence.<ns:*>); // @output <ns:root xmlns:ns="foo"/>
    io:println(sequence.<k:*>); // @output <k:root xmlns:k="bar"/><k:item xmlns:k="bar"/>
    io:println(sequence.<ns:*|k:*>); // @output <ns:root xmlns:ns="foo"/><k:root xmlns:k="bar"/><k:item xmlns:k="bar"/>
    io:println(sequence.<ns:root|k:root>); // @output <ns:root xmlns:ns="foo"/><k:root xmlns:k="bar"/>
    io:println(sequence.<ns:other|k:item>); // @output <k:item xmlns:k="bar"/>
    io:println(sequence.<ns:*|k:*>.<ns:root|k:item>); // @output <ns:root xmlns:ns="foo"/><k:item xmlns:k="bar"/>

    xml mixed = xml `<a/>text<!--comment--><?target data?>`;
    io:println(mixed.<*>); // @output <a/>

    xmlns "urn:shared" as p;
    xmlns "urn:shared" as q;
    xml aliasItem = xml `<p:item></p:item>`;
    xml noNamespaceItem = xml `<item></item>`;
    xmlns "urn:default";
    xml defaultItem = xml `<item></item>`;
    io:println(defaultItem.<item>); // @output <item xmlns="urn:default"/>
    io:println(noNamespaceItem.<item>); // @output
    io:println(aliasItem.<q:item>); // @output <p:item xmlns:p="urn:shared"/>
    io:println(aliasItem.<q:item|p:*>); // @output <p:item xmlns:p="urn:shared"/>

    xml immediate = xml `<parent><target></target></parent><target></target>`;
    io:println(immediate.<target>); // @output <target xmlns="urn:default"/>
}
