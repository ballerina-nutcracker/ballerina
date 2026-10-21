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

import ballerina/data.jsondata;
import ballerina/io;

type Address record {|
    string city;
    int zip;
|};

type Person record {|
    string name;
    int age;
    float height;
    Address address;
    string[] tags;
    string? nickname;
    int score?;
|};

type OpenRec record {
    string id;
};

type IntRest record {|
    string id;
    int...;
|};

type Point [int, int];

type Book record {|
    @jsondata:Name {value: "book-title"}
    string title;
    @jsondata:Name {value: "author-name"}
    string author;
|};

public function main() returns error? {
    json personJson = {
        name: "Alice",
        age: 30,
        height: 165,
        address: {city: "Colombo", zip: 10100},
        tags: ["a", "b"],
        nickname: null
    };
    Person p = check jsondata:parseAsType(personJson);
    io:println(p.name, " ", p.age, " ", p.height, " ", p.address.city, " ", p.tags.length(), " ", p.nickname is ());
    // @output Alice 30 165.0 Colombo 2 true

    string src = string `{"name":"Bob","age":25,"height":1.8e0,"address":{"city":"Kandy","zip":20000},"tags":[],"nickname":"bobby","score":7}`;
    Person q = check jsondata:parseString(src);
    io:println(q.name, " ", q.height, " ", q.nickname, " ", q.score);
    // @output Bob 1.8 bobby 7

    Person r = check jsondata:parseBytes(src.toBytes());
    io:println(r.address.zip);
    // @output 20000

    stream<byte[], error?> byteStream = [src.substring(0, 10).toBytes(), src.substring(10).toBytes()].toStream();
    Person s = check jsondata:parseStream(byteStream);
    io:println(s.age);
    // @output 25

    record {|string name;|} projected = check jsondata:parseString(src);
    io:println(projected);
    // @output {"name":"Bob"}

    record {|string name;|}|jsondata:Error strict = jsondata:parseString(src, {allowDataProjection: false});
    io:println(strict is jsondata:Error);
    // @output true

    record {|string name; int age?;|} nilOpt = check jsondata:parseString(string `{"name":"C","age":null}`,
            {allowDataProjection: {nilAsOptionalField: true}});
    io:println(nilOpt);
    // @output {"name":"C"}

    record {|string name; string? alias;|} absentNil = check jsondata:parseString(string `{"name":"D"}`,
            {allowDataProjection: {absentAsNilableType: true}});
    io:println(absentNil);
    // @output {"name":"D","alias":null}

    OpenRec open = check jsondata:parseString(string `{"id":"x","extra":{"k":1}}`);
    io:println(open);
    // @output {"id":"x","extra":{"k":1}}

    IntRest rest = check jsondata:parseAsType({id: "y", a: 1, b: 2});
    io:println(rest);
    // @output {"id":"y","a":1,"b":2}

    Point pt = check jsondata:parseString("[1, 2, 3]");
    io:println(pt);
    // @output [1,2]

    int[2] closed = check jsondata:parseString("[4, 5, 6]");
    io:println(closed);
    // @output [4,5]

    map<int> m = check jsondata:parseString(string `{"a":1,"b":2}`);
    io:println(m);
    // @output {"a":1,"b":2}

    int|string u = check jsondata:parseString(string `"hello"`);
    io:println(u);
    // @output hello

    decimal d = check jsondata:parseString("12.50");
    io:println(d);
    // @output 12.50

    json anyJson = check jsondata:parseString(string `{"n":1,"f":1.5,"b":true,"z":null}`);
    io:println(anyJson);
    // @output {"n":1,"f":1.5,"b":true,"z":null}

    readonly & Address ro = check jsondata:parseString(string `{"city":"Galle","zip":80000}`);
    io:println(ro is readonly);
    // @output true

    Book book = check jsondata:parseString(string `{"book-title":"Dune","author-name":"Herbert"}`, {}, Book);
    io:println(book.title, " by ", book.author);
    // @output Dune by Herbert

    int|jsondata:Error bad = jsondata:parseString(string `"abc"`);
    if bad is jsondata:Error {
        io:println(bad.message());
    }
    // @output incompatible expected type 'int' for value 'abc'

    Person|jsondata:Error missing = jsondata:parseAsType({name: "E", height: 1.0, address: {city: "X", zip: 1}, tags: [], nickname: ()});
    if missing is jsondata:Error {
        io:println(missing.message());
    }
    // @output required field 'age' not present in JSON

    json|jsondata:Error syntax = jsondata:parseString("{\"a\" 1}");
    if syntax is jsondata:Error {
        io:println(syntax.message());
    }
    // @output 'expected ':'' at line: '1' column: '6'

    json converted = jsondata:toJson({name: "F", items: [1, 2], nested: {ok: true}});
    io:println(converted);
    // @output {"name":"F","items":[1,2],"nested":{"ok":true}}
}
