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

enum Color {
    RED,
    GREEN
}

service /q on new http:Listener(19230) {
    resource function get basic(string foo, int? bar, int[] ids, decimal d, boolean b, float f) returns string {
        return string `foo=${foo} bar=${bar.toString()} ids=${ids.toString()} d=${d} b=${b} f=${f}`;
    }

    resource function get str(string foo) returns string {
        return string `[${foo}]`;
    }

    resource function get strs(string[] foo) returns string {
        return foo.toString();
    }

    resource function get opt(string? foo) returns string {
        return foo is () ? "nil" : string `[${foo}]`;
    }

    resource function get optarr(int[]? ids) returns string {
        return ids is () ? "nil" : ids.toString();
    }

    resource function get def(int id = 10) returns string {
        return id.toString();
    }

    resource function get color(Color c) returns string {
        return c;
    }

    resource function get m(@http:Query map<json> q) returns json {
        return q;
    }

    resource function get rec(Person p) returns string {
        return string `${p.name} ${p.age}`;
    }

    resource function get recs(@http:Query Person[] ps) returns int {
        return ps.length();
    }

    resource function get named(@http:Query {name: "the-key"} string k) returns string {
        return k;
    }

    resource function get kw(string 'type) returns string {
        return 'type;
    }

    resource function get spaced(@http:Query {name: "a b"} string v) returns string {
        return v;
    }

    resource function get num(int n) returns string {
        return n.toString();
    }

    resource function get fl(float n) returns string {
        return n.toString();
    }

    resource function get ro(readonly & int[] n) returns string {
        return (n is readonly).toString();
    }

    resource function get [int id](string label, http:Request req) returns string {
        return string `${id} ${label} ${req.getQueryParamValue("label") ?: "-"}`;
    }

    resource function get badtype(xml x) returns string {
        return x.toString();
    }

    resource function get badunion(int|string p) returns string {
        return p.toString();
    }

    resource function get tworeq(http:Request a, http:Request b) returns string {
        return a.rawPath + b.rawPath;
    }
}

function show(http:Client c, string path) returns string|error {
    http:Response r = check c->get(path);
    if r.statusCode >= 400 {
        json body = check r.getJsonPayload();
        return string `${r.statusCode} ${(check body.message).toString()}`;
    }
    return string `${r.statusCode} ${check r.getTextPayload()}`;
}

public function testMain() returns error? {
    http:Client c = check new ("http://localhost:19230/q");
    io:println(show(c, "/basic?foo=a+b&ids=1,2&ids=3&ids=1&d=1.50&b=TRUE&f=1e2&bar")); // @output 200 foo=a b bar= ids=[1,2,3,1] d=1.50 b=true f=100.0
    io:println(show(c, "/basic?ids=1&d=1&b=true&f=1")); // @output 400 no query param value found for 'foo'
    io:println(show(c, "/basic?foo=a&ids=x&d=1&b=true&f=1")); // @output 400 error in casting query param : 'ids'
    io:println(show(c, "/basic?foo=a&ids=1&d=1&b=yes&f=1")); // @output 200 foo=a bar= ids=[1] d=1 b=false f=1.0
    io:println(show(c, "/str?foo=")); // @output 200 []
    io:println(show(c, "/str?foo")); // @output 400 no query param value found for 'foo'
    io:println(show(c, "/str?foo=a,b")); // @output 200 [a]
    io:println(show(c, "/str?foo=x&foo=y")); // @output 200 [x]
    io:println(show(c, "/str?fo%6F=a")); // @output 400 no query param value found for 'foo'
    io:println(show(c, "/str?foo=a%2Cb")); // @output 200 [a,b]
    io:println(show(c, "/str?foo=%zz")); // @output 500 invalid URL escape "%zz"
    io:println(show(c, "/strs?foo=a,b&foo=a&foo=c")); // @output 200 ["a","b","a","c"]
    io:println(show(c, "/opt?foo")); // @output 200 nil
    io:println(show(c, "/opt")); // @output 200 nil
    io:println(show(c, "/opt?foo=")); // @output 200 []
    io:println(show(c, "/optarr")); // @output 200 nil
    io:println(show(c, "/optarr?ids=4,5")); // @output 200 [4,5]
    io:println(show(c, "/def?id=5")); // @output 200 5
    // The runtime does not expose the default value, so a missing defaultable parameter is rejected.
    io:println(show(c, "/def")); // @output 400 no query param value found for 'id'
    io:println(show(c, "/color?c=RED")); // @output 200 RED
    io:println(show(c, "/color?c=BLUE")); // @output 400 error in casting query param : 'c'
    io:println(show(c, "/m?q=%7B%22a%22%3A1%7D")); // @output 200 {"a":1}
    io:println(show(c, "/rec?p=%7B%22name%22%3A%22a%22%2C%22age%22%3A1%7D")); // @output 200 a 1
    io:println(show(c, "/rec?p=%7B%22name%22%3A%22a%22%7D")); // @output 400 error in casting query param : 'p'
    io:println(show(c, "/rec?p=%7Bbad")); // @output 400 error in casting query param : 'p'
    io:println(show(c, "/recs?ps=%7B%22name%22%3A%22a%22%2C%22age%22%3A1%7D&ps=%7B%22name%22%3A%22b%22%2C%22age%22%3A2%7D")); // @output 200 2
    io:println(show(c, "/named?the-key=v")); // @output 200 v
    io:println(show(c, "/kw?type=t")); // @output 200 t
    io:println(show(c, "/spaced?a%20b=v")); // @output 200 v
    io:println(show(c, "/num?n=%2B5")); // @output 200 5
    io:println(show(c, "/num?n=%205")); // @output 400 error in casting query param : 'n'
    io:println(show(c, "/num?n=0x10")); // @output 400 error in casting query param : 'n'
    io:println(show(c, "/fl?n=Infinity")); // @output 200 Infinity
    io:println(show(c, "/fl?n=-Infinity")); // @output 200 -Infinity
    io:println(show(c, "/fl?n=NaN")); // @output 200 NaN
    io:println(show(c, "/fl?n=Inf")); // @output 400 error in casting query param : 'n'
    io:println(show(c, "/fl?n=1.5d")); // @output 200 1.5
    io:println(show(c, "/fl?n=1_0")); // @output 400 error in casting query param : 'n'
    io:println(show(c, "/ro?n=1,2")); // @output 200 true
    io:println(show(c, "/7?label=x")); // @output 200 7 x x
    io:println(show(c, "/badtype?x=1")); // @output 500 invalid type of query param x: expected one of the string, int, float, boolean, decimal, map<anydata> types or the array types of them
    io:println(show(c, "/badunion?p=1")); // @output 500 invalid type of query param p: expected one of the string, int, float, boolean, decimal, map<anydata> types or the array types of them
    io:println(show(c, "/tworeq")); // @output 500 invalid multiple http:Request parameter: b
}
