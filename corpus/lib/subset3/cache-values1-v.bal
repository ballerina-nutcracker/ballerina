// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

import ballerina/cache;
import ballerina/io;

type Person record {|
    string name;
    int age;
|};

class Greeter {
    function greet() returns string {
        return "hello";
    }
}

class MapCache {
    *cache:AbstractCache;

    private map<string> entries = {};

    public isolated function put(string key, any value, decimal maxAge = -1) returns cache:Error? {
        if value !is string || maxAge < -1d {
            return error cache:Error("only strings: " + key);
        }
        self.entries[key] = value;
    }

    public isolated function get(string key) returns any|cache:Error {
        string? v = self.entries[key];
        return v ?: error cache:Error("missing: " + key);
    }

    public isolated function invalidate(string key) returns cache:Error? {
        if self.entries.hasKey(key) {
            _ = self.entries.remove(key);
        }
    }

    public isolated function invalidateAll() returns cache:Error? {
        self.entries = {};
    }

    public isolated function hasKey(string key) returns boolean {
        return self.entries.hasKey(key);
    }

    public isolated function keys() returns string[] {
        return self.entries.keys();
    }

    public isolated function size() returns int {
        return self.entries.length();
    }

    public isolated function capacity() returns int {
        return -1;
    }
}

function describe(cache:AbstractCache c) returns error? {
    check c.put("k", "v");
    io:println(c.hasKey("k"), " ", check c.get("k"), " ", c.size(), " ", c.capacity());
    check c.invalidate("k");
    io:println(c.hasKey("k"), " ", c.keys());
}

public function main() returns error? {
    cache:Cache c = new;
    check c.put("int", 42);
    check c.put("float", 1.5);
    check c.put("decimal", 2.5d);
    check c.put("string", "str");
    check c.put("boolean", false);
    check c.put("list", [1, "two"]);
    check c.put("map", {a: 1});
    check c.put("record", <Person>{name: "Ann", age: 30});
    check c.put("object", new Greeter());
    check c.put("", "empty key");

    io:println(check c.get("int"), " ", check c.get("float"), " ", check c.get("decimal")); // @output 42 1.5 2.5
    io:println(check c.get("string"), " ", check c.get("boolean"), " ", check c.get("")); // @output str false empty key
    io:println(check c.get("list"), " ", check c.get("map")); // @output [1,"two"] {"a":1}

    any rec = check c.get("record");
    if rec is Person {
        io:println(rec.name, " ", rec.age); // @output Ann 30
    }
    any obj = check c.get("object");
    if obj is Greeter {
        io:println(obj.greet()); // @output hello
    }

    int[] shared = [1];
    check c.put("shared", shared);
    shared.push(2);
    io:println(check c.get("shared")); // @output [1,2]

    check c.put("int", 43);
    io:println(check c.get("int"), " ", c.size()); // @output 43 11

    cache:Error? missing = c.invalidate("nope");
    io:println(missing is cache:Error ? missing.message() : "none"); // @output Cache entry from the given key: nope, is not available.
    check c.invalidate("");
    io:println(c.hasKey(""), " ", c.size()); // @output false 10

    check describe(c);
    // @output true v 11 100
    // @output false ["float","decimal","string","boolean","list","map","record","object","shared","int"]
    check describe(new MapCache());
    // @output true v 1 -1
    // @output false []
    cache:Error? e = new MapCache().put("x", 1);
    io:println(e is cache:Error ? e.message() : "none"); // @output only strings: x
}
