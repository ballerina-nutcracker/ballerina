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

import ballerina/http;
import ballerina/io;

type Person record {|
    string name;
    int age;
|};

type OpenPerson record {
    string name;
};

enum Color {
    RED,
    GREEN
}

service /p on new http:Listener(19232) {
    resource function post rec(@http:Payload Person p) returns string {
        return string `${p.name} ${p.age}`;
    }

    resource function post defrec(Person p) returns string {
        return string `${p.name} ${p.age}`;
    }

    resource function post open(OpenPerson p) returns string {
        return string `${p.name} ${p["extra"].toString()}`;
    }

    resource function post str(@http:Payload string s) returns string {
        return string `[${s}]`;
    }

    resource function post jsonopt(@http:Payload json? j) returns string {
        return j is () ? "nil" : j.toString();
    }

    resource function post recopt(Person? p) returns string {
        return p is () ? "nil" : p.name;
    }

    resource function post strempty(@http:Payload string? s) returns string {
        return s is () ? "nil" : string `[${s}]`;
    }

    resource function post mt(@http:Payload {mediaType: "application/xml"} xml x) returns string {
        return x.toString();
    }

    resource function post form(@http:Payload map<string> m) returns string {
        return string `${m["a"] ?: "-"} ${m["b"] ?: "-"}`;
    }

    resource function post bytes(@http:Payload byte[] b) returns int {
        return b.length();
    }

    resource function post intp(@http:Payload int i) returns string {
        return i.toString();
    }

    resource function post union(@http:Payload json|xml v) returns string {
        return v is xml ? "xml " + v.toString() : "json " + v.toString();
    }

    resource function post xmlp(xml v) returns string {
        return v.toString();
    }

    resource function post arr(Person[] ps) returns int {
        return ps.length();
    }

    resource function post withq(Person p, string q) returns string {
        return string `${p.name} ${q}`;
    }

    resource function post intarr(int[] q) returns string {
        return q.toString();
    }

    resource function post bytesdef(byte[] b) returns int {
        return b.length();
    }

    resource function post ro(readonly & Person p) returns string {
        return (p is readonly).toString();
    }

    resource function post color(@http:Payload Color c) returns string {
        return c;
    }

    resource function put withreq(Person p, http:Request req) returns string|error {
        json body = check req.getJsonPayload();
        return string `${p.name} ${(check body.age).toString()}`;
    }

    resource function patch patched(map<json> m) returns string {
        return m.toString();
    }

    resource function delete deleted(@http:Payload string s) returns string {
        return s;
    }

    resource function default anything(Person p) returns string {
        return p.name;
    }

    resource function get getrec(Person p) returns string {
        return p.name;
    }

    resource function post amb(Person a, map<json> b) returns string {
        return a.name + b.toString();
    }

    resource function post mixed(Person|string p) returns string {
        return p.toString();
    }

    resource function post qunion(int|string p) returns string {
        return p.toString();
    }

    resource function post two(@http:Payload Person a, @http:Payload Person b) returns string {
        return a.name + b.name;
    }

    resource function get getpayload(@http:Payload Person a) returns string {
        return a.name;
    }
}

function show(http:Response r) returns string|error {
    if r.statusCode >= 400 {
        json body = check r.getJsonPayload();
        return string `${r.statusCode} ${(check body.message).toString()}`;
    }
    return string `${r.statusCode} ${check r.getTextPayload()}`;
}

function post(http:Client c, string path, http:RequestMessage message, string? mediaType = ()) returns string|error {
    http:Response r = check c->post(path, message, mediaType = mediaType);
    return show(r);
}

function raw(http:Client c, string method, string path, string contentType, string body = "") returns string|error {
    http:Request req = new;
    if contentType != "" {
        req.setHeader("Content-Type", contentType);
    }
    if body != "" {
        req.setTextPayload(body, contentType == "" ? () : contentType);
        if contentType == "" {
            req.removeHeader("Content-Type");
        }
    }
    http:Response r = check c->execute(method, path, req);
    return show(r);
}

public function testMain() returns error? {
    http:Client c = check new ("http://localhost:19232/p");
    io:println(post(c, "/rec", {name: "a", age: 1})); // @output 201 a 1
    // Conversion failures carry lang.value:cloneWithType wording, and types print structurally.
    io:println(post(c, "/rec", {name: "a", age: 1, extra: true})); // @output 400 data binding failed: 'true' value cannot be converted to 'never'
    io:println(post(c, "/rec", {name: "a"})); // @output 400 data binding failed: '{| json... |}' value cannot be converted to '{| age: int, name: string, never... |}': field 'age' not present in value
    io:println(post(c, "/rec", "hello")); // @output 400 data binding failed: incompatible type found: '{| age: int, name: string, never... |}'
    io:println(post(c, "/rec", xml `<a/>`)); // @output 400 data binding failed: incompatible type found: '{| age: int, name: string, never... |}'
    io:println(post(c, "/rec", "{}".toBytes())); // @output 400 data binding failed: incompatible type found: '{| age: int, name: string, never... |}'
    io:println(raw(c, "POST", "/rec", "application/json", "{bad")); // @output 400 data binding failed: failed to parse JSON payload: invalid character 'b' looking for beginning of object key string
    io:println(raw(c, "POST", "/xmlp", "application/xml", "<a>")); // @output 400 data binding failed: failed to parse xml: unexpected end of XML content
    io:println(raw(c, "POST", "/form", "application/x-www-form-urlencoded", "a=%zz")); // @output 400 data binding failed: invalid URL escape "%zz"
    io:println(post(c, "/color", "BLUE")); // @output 400 data binding failed: '"BLUE"' value cannot be converted to '"GREEN"|"RED"'
    io:println(post(c, "/defrec", {name: "b", age: 2})); // @output 201 b 2
    io:println(post(c, "/open", {name: "a", extra: 1})); // @output 201 a 1
    io:println(post(c, "/str", "hello")); // @output 201 [hello]
    io:println(raw(c, "POST", "/str", "", "plain")); // @output 201 [plain]
    io:println(raw(c, "POST", "/str", "text/plain; charset=utf-8", "hi")); // @output 201 [hi]
    io:println(raw(c, "POST", "/str", "application/vnd.foo+json", "\"x\"")); // @output 201 [x]
    io:println(raw(c, "POST", "/str", "application/unknown", "raw")); // @output 201 [raw]
    io:println(raw(c, "POST", "/str", "text/plain")); // @output 400 data binding failed: String payload is null
    io:println(raw(c, "POST", "/jsonopt", "application/json")); // @output 201 nil
    io:println(raw(c, "POST", "/jsonopt", "")); // @output 201 nil
    io:println(raw(c, "POST", "/recopt", "application/json")); // @output 201 nil
    io:println(raw(c, "POST", "/recopt", "")); // @output 201 nil
    io:println(raw(c, "POST", "/strempty", "text/plain")); // @output 201 nil
    io:println(raw(c, "POST", "/rec", "application/json")); // @output 400 data binding failed: empty JSON document
    io:println(raw(c, "POST", "/rec", "")); // @output 400 data binding failed: empty JSON document
    io:println(raw(c, "POST", "/rec", "application/unknown", "{\"name\":\"a\",\"age\":1}")); // @output 201 a 1
    io:println(raw(c, "POST", "/xmlp", "application/xml")); // @output 400 data binding failed: Empty xml payload
    io:println(raw(c, "POST", "/xmlp", "", "<a/>")); // @output 201 <a/>
    io:println(raw(c, "POST", "/bytes", "application/octet-stream")); // @output 201 0
    io:println(raw(c, "POST", "/form", "application/x-www-form-urlencoded")); // @output 400 data binding failed: String payload is null
    io:println(post(c, "/mt", xml `<a/>`)); // @output 201 <a/>
    io:println(post(c, "/form", "a=1&b=x+y&a=2", "application/x-www-form-urlencoded")); // @output 201 2 x y
    io:println(raw(c, "POST", "/form", "text/plain", "a=1")); // @output 201 1 -
    io:println(post(c, "/bytes", "[1,2,3]".toBytes())); // @output 201 7
    io:println(post(c, "/bytes", "abcd")); // @output 201 4
    io:println(raw(c, "POST", "/bytes", "application/json", "[1,2]")); // @output 201 2
    io:println(raw(c, "POST", "/bytes", "application/unknown", "raw")); // @output 201 3
    io:println(post(c, "/intp", "5")); // @output 400 data binding failed: incompatible type found: 'int'
    io:println(post(c, "/intp", 5)); // @output 201 5
    io:println(post(c, "/union", xml `<a>1</a>`)); // @output 201 xml <a>1</a>
    io:println(post(c, "/union", {a: 1})); // @output 201 json {"a":1}
    io:println(post(c, "/xmlp", xml `<b/>`)); // @output 201 <b/>
    io:println(post(c, "/arr", [{name: "a", age: 1}, {name: "b", age: 2}])); // @output 201 2
    io:println(post(c, "/withq?q=z", {name: "a", age: 1})); // @output 201 a z
    io:println(post(c, "/intarr?q=1,2", {name: "a", age: 1})); // @output 201 [1,2]
    io:println(post(c, "/bytesdef", "xyz")); // @output 201 3
    io:println(post(c, "/ro", {name: "a", age: 1})); // @output 201 true
    io:println(post(c, "/color", "RED")); // @output 201 RED
    io:println(raw(c, "PUT", "/withreq", "application/json", "{\"name\":\"a\",\"age\":1}")); // @output 200 a 1
    io:println(raw(c, "PATCH", "/patched", "application/json", "{\"k\":true}")); // @output 200 {"k":true}
    io:println(raw(c, "DELETE", "/deleted", "text/plain", "gone")); // @output 200 gone
    io:println(raw(c, "POST", "/anything", "application/json", "{\"name\":\"d\",\"age\":1}")); // @output 200 d
    io:println(raw(c, "GET", "/getrec?p=%7B%22name%22%3A%22q%22%2C%22age%22%3A1%7D", "")); // @output 200 q
    io:println(post(c, "/amb", {name: "a", age: 1})); // @output 500 ambiguous types for parameter a and b. Use annotations to avoid ambiguity
    io:println(post(c, "/mixed", "x")); // @output 500 invalid union type for default payload param: p. Use basic structured anydata types
    io:println(post(c, "/qunion?p=1", "x")); // @output 500 invalid type of query param p: expected one of the string, int, float, boolean, decimal, map<anydata> types or the array types of them
    io:println(post(c, "/two", {name: "a", age: 1})); // @output 500 invalid multiple 'http:Payload' annotation usage
    io:println(raw(c, "GET", "/getpayload", "")); // @output 500 invalid usage of payload annotation for a non entity body resource : get. Use an accessor that supports entity body
}
