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

type Outer record {|
    string s = "outer";
|};

type SecondOuter record {|
    int n = 1;
|};

class Counter {
    function localDefault() returns int {
        record {|int m = 7;|} local = {};
        return local.m;
    }

    function paramDefault(record {|int k = 3;|} rec) returns int {
        return rec.k;
    }
}

function requiredRecordParam(record {|string v = "required";|} rec) returns string {
    return rec.v;
}

function defaultableRecordParam(record {|int w = 11;|} rec = {}) returns int {
    return rec.w;
}

function multipleRecordParams(record {|string x = "first";|} a,
        record {|string y = "second";|} b) returns string {
    return a.x + "-" + b.y;
}

function firstLocal() returns string {
    record {|string t = "local";|} local = {};
    return local.t;
}

function secondLocal() returns int {
    record {|int u = 42;|} local = {};
    return local.u;
}

function lambdaDefaults() returns string {
    var parent = function() returns string {
        record {|string p = "parent";|} parentRecord = {};
        var nested = function() returns string {
            record {|string q = "nested";|} nestedRecord = {};
            return nestedRecord.q;
        };
        var sibling = function() returns string {
            record {|string r = "sibling";|} siblingRecord = {};
            return siblingRecord.r;
        };
        return parentRecord.p + "-" + nested() + "-" + sibling();
    };
    return parent();
}

public function main() {
    Outer o = {};
    io:println(o.s); // @output outer
    io:println(firstLocal()); // @output local

    SecondOuter so = {};
    io:println(so.n); // @output 1
    io:println(secondLocal()); // @output 42

    io:println(lambdaDefaults()); // @output parent-nested-sibling

    Counter c = new;
    io:println(c.localDefault()); // @output 7
    io:println(c.paramDefault({})); // @output 3

    io:println(requiredRecordParam({})); // @output required
    io:println(defaultableRecordParam()); // @output 11
    io:println(multipleRecordParams({}, {})); // @output first-second
}
