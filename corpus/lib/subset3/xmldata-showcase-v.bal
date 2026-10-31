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

import ballerina/data.xmldata;
import ballerina/io;

type Author record {|
    string name;
    string country;
|};

type Title record {|
    string \#content;
    string lang;
|};

type Book record {|
    int id;
    Title title;
    Author author;
    string[] tag;
    decimal price;
    boolean available;
    string isbn;
|};

type Summary record {|
    int id;
|};

type Catalog record {
    string name;
};

@xmldata:Name {value: "item"}
type Item record {|
    @xmldata:Attribute
    string code;
    @xmldata:Name {value: "item-name"}
    string itemName;
|};

@xmldata:Namespace {prefix: "bk", uri: "http://example.com/book"}
type NsBook record {|
    @xmldata:Namespace {prefix: "bk", uri: "http://example.com/book"}
    string title;
|};

type Prefixed record {|
    string attr_lang;
    string text;
|};

public function main() returns error? {
    string bookXml = string `<book isbn="978-0">
        <id>601970</id>
        <title lang="en">Dune</title>
        <author><name>Herbert</name><country>US</country></author>
        <tag>scifi</tag>
        <tag>classic</tag>
        <price>10.50</price>
        <available>true</available>
    </book>`;
    Book book = check xmldata:parseString(bookXml);
    io:println(book.id, " ", book.title.\#content, " ", book.title.lang, " ", book.author.name, " ", book.tag, " ",
            book.price, " ", book.available, " ", book.isbn);
    // @output 601970 Dune en Herbert ["scifi","classic"] 10.50 true 978-0

    Book fromBytes = check xmldata:parseBytes(bookXml.toBytes());
    io:println(fromBytes.author.country);
    // @output US

    stream<byte[], error?> chunks = [bookXml.substring(0, 20).toBytes(), bookXml.substring(20).toBytes()].toStream();
    Book fromStream = check xmldata:parseStream(chunks);
    io:println(fromStream.tag.length());
    // @output 2

    xml bookValue = xml `<book><id>7</id><extra>x</extra></book>`;
    Summary summary = check xmldata:parseAsType(bookValue);
    io:println(summary);
    // @output {"id":7}

    Summary|xmldata:Error strict = xmldata:parseAsType(bookValue, {allowDataProjection: false});
    io:println(strict is xmldata:Error);
    // @output true

    Catalog catalog = check xmldata:parseString(string `<catalog><name>Main</name><owner>Ann</owner><count>3</count></catalog>`);
    io:println(catalog);
    // @output {"name":"Main","owner":"Ann","count":3}

    Item item = check xmldata:parseString(string `<item code="A1"><item-name>Pen</item-name></item>`, {}, Item);
    io:println(item);
    // @output {"code":"A1","itemName":"Pen"}

    NsBook nsBook = check xmldata:parseString(
            string `<bk:book xmlns:bk="http://example.com/book"><bk:title>Emma</bk:title></bk:book>`, {}, NsBook);
    io:println(nsBook);
    // @output {"title":"Emma"}

    Prefixed prefixed = check xmldata:parseString(string `<p lang="si">hello</p>`,
            {attributePrefix: "attr_", textFieldName: "text"});
    io:println(prefixed);
    // @output {"attr_lang":"si","text":"hello"}

    Summary|xmldata:Error bad = xmldata:parseString("<book><id>x</id></book>");
    if bad is xmldata:Error {
        io:println(bad.message());
    }
    // @output 'string' value 'x' cannot be converted to 'int'

    xmldata:Error? valid = xmldata:validate(xml `<item code="A1"><item-name>Pen</item-name></item>`, Item);
    io:println(valid is ());
    // @output true
    xmldata:Error? invalid = xmldata:validate(xml `<item><item-name>Pen</item-name></item>`, Item);
    io:println(invalid is xmldata:Error);
    // @output true
}
