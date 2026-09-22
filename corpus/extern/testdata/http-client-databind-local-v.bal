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

type Address record {|
    string city;
    int zip;
|};

type Employee record {|
    string name;
    Address address;
|};

// A closed all-string record is a subtype of map<string>, so the form builder is selected.
type Form record {|
    string a;
    string b;
|};

// A tuple is a subtype of byte[], so the blob builder is selected for it.
type Pair [byte, byte];

enum Colour {
    RED = "red",
    GREEN = "green"
}

// Each resource serves one body with the Content-Type that selects a particular payload
// builder. Responses a conforming service cannot produce — one with no Content-Type at all —
// are covered by http-client-databind-no-content-type-local-v.bal instead.
service /db on new http:Listener(19220) {
    resource function get person() returns http:Response {
        http:Response r = new;
        r.setJsonPayload({name: "Alice", age: 30});
        return r;
    }

    resource function get personHal() returns http:Response {
        http:Response r = new;
        r.setJsonPayload({name: "Alice", age: 30}, "application/hal+json");
        return r;
    }

    resource function get employee() returns http:Response {
        http:Response r = new;
        r.setJsonPayload({name: "Alice", address: {city: "Colombo", zip: 100}});
        return r;
    }

    resource function get people() returns http:Response {
        http:Response r = new;
        r.setJsonPayload([{name: "Alice", age: 30}, {name: "Bob", age: 25}]);
        return r;
    }

    resource function get count() returns http:Response {
        http:Response r = new;
        r.setJsonPayload(7);
        return r;
    }

    resource function get flag() returns http:Response {
        http:Response r = new;
        r.setJsonPayload(true);
        return r;
    }

    resource function get text() returns http:Response {
        http:Response r = new;
        r.setTextPayload("plain text body");
        return r;
    }

    resource function get colour() returns http:Response {
        http:Response r = new;
        r.setTextPayload("red");
        return r;
    }

    resource function get blob() returns http:Response {
        http:Response r = new;
        r.setBinaryPayload([1, 2, 3]);
        return r;
    }

    // Two bytes, the length a [byte, byte] tuple target accepts.
    resource function get blob2() returns http:Response {
        http:Response r = new;
        r.setBinaryPayload([1, 2]);
        return r;
    }

    resource function get form() returns http:Response {
        http:Response r = new;
        r.setTextPayload("a=1&b=two", "application/x-www-form-urlencoded");
        return r;
    }

    resource function get unknown() returns http:Response {
        http:Response r = new;
        r.setTextPayload("opaque payload", "application/vnd.custom");
        return r;
    }

    resource function get empty() returns http:Response {
        http:Response r = new;
        r.setTextPayload("");
        return r;
    }

    resource function get emptyForm() returns http:Response {
        http:Response r = new;
        r.setTextPayload("", "application/x-www-form-urlencoded");
        return r;
    }

    resource function get emptyBlob() returns http:Response {
        http:Response r = new;
        byte[] none = [];
        r.setBinaryPayload(none);
        return r;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19220", {});

    // application/json → record.
    Person p = check c->get("/db/person");
    io:println(p.name, " ", p.age); // @output Alice 30

    // application/json → nested record.
    Employee e = check c->get("/db/employee");
    io:println(e.address.city); // @output Colombo

    // application/json → record array.
    Person[] people = check c->get("/db/people");
    io:println(people.length(), " ", people[1].name); // @output 2 Bob

    // application/json → map<json>.
    map<json> asMap = check c->get("/db/person");
    io:println(asMap["name"]); // @output Alice

    // application/json → json. Map member order is not fixed, so only the shape is asserted.
    json asJson = check c->get("/db/person");
    io:println(asJson is map<json>); // @output true

    // application/json → scalar targets.
    int count = check c->get("/db/count");
    io:println(count); // @output 7
    boolean flag = check c->get("/db/flag");
    io:println(flag); // @output true

    // A json media type with a suffix still selects the json builder.
    Person suffixed = check c->get("/db/personHal");
    io:println(suffixed.name); // @output Alice

    // text/plain → string.
    string text = check c->get("/db/text");
    io:println(text); // @output plain text body

    // text/plain → byte[].
    byte[] textBytes = check c->get("/db/text");
    io:println(textBytes.length()); // @output 15

    // text/plain → nilable byte[].
    byte[]? optTextBytes = check c->get("/db/text");
    io:println(optTextBytes is byte[]); // @output true

    // application/octet-stream → byte[].
    byte[] blob = check c->get("/db/blob");
    io:println(blob.length(), " ", blob[0]); // @output 3 1

    // application/octet-stream → nilable byte[].
    byte[]? optBlob = check c->get("/db/blob");
    io:println(optBlob is byte[]); // @output true

    // application/x-www-form-urlencoded → map<string>.
    map<string> form = check c->get("/db/form");
    io:println(form["a"], " ", form["b"]); // @output 1 two

    // application/x-www-form-urlencoded → nilable map<string>.
    map<string>? optForm = check c->get("/db/form");
    io:println(optForm is map<string>); // @output true

    // application/x-www-form-urlencoded → string keeps the raw body.
    string rawForm = check c->get("/db/form");
    io:println(rawForm); // @output a=1&b=two

    // application/x-www-form-urlencoded → nilable string.
    string? optRawForm = check c->get("/db/form");
    io:println(optRawForm); // @output a=1&b=two

    // An unknown media type falls back to the target type.
    string unknown = check c->get("/db/unknown");
    io:println(unknown); // @output opaque payload

    // A target narrower than the builder's own type is converted to that target.
    Form formRecord = check c->get("/db/form");
    io:println(formRecord.a, " ", formRecord.b); // @output 1 two
    io:println(<any>formRecord is Form); // @output true

    Colour colour = check c->get("/db/colour");
    io:println(colour, " ", <any>colour is Colour); // @output red true

    Pair pair = check c->get("/db/blob2");
    io:println(pair.length(), " ", <any>pair is Pair); // @output 2 true

    // The nilable form reaches the same builder; an absent body binds to ().
    Colour? optColour = check c->get("/db/colour");
    io:println(optColour, " ", <any>optColour is Colour); // @output red true
    Colour? noColour = check c->get("/db/empty");
    io:println(noColour is ()); // @output true

    Form? optFormRecord = check c->get("/db/form");
    io:println(optFormRecord?.a, " ", <any>optFormRecord is Form); // @output 1 true
    Form? noFormRecord = check c->get("/db/emptyForm");
    io:println(noFormRecord is ()); // @output true

    Pair? optPair = check c->get("/db/blob2");
    io:println(<any>optPair is Pair); // @output true
    Pair? noPair = check c->get("/db/emptyBlob");
    io:println(noPair is ()); // @output true

    // `var` provides no contextually expected type, so the target is passed explicitly.
    var explicitPerson = check c->get("/db/person", targetType = Person);
    io:println(explicitPerson.name); // @output Alice

    var explicitResponse = check c->get("/db/person", targetType = http:Response);
    io:println(explicitResponse.statusCode); // @output 200

    return;
}
