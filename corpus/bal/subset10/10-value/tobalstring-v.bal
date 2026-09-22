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

type Address record {|
    string city;
    string country;
|};

class Counter {
    int n = 0;
    public isolated function next() returns record {|int value;|}|error? {
        int current = self.n;
        if current >= 3 {
            return ();
        }
        self.n = current + 1;
        return {value: current};
    }
}

public function main() {
    int i = 12;
    io:println(i.toBalString()); // @output 12

    string s = "Anne";
    io:println(s.toBalString()); // @output "Anne"

    string quoted = "she said \"hi\" \\ ok";
    io:println(quoted.toBalString()); // @output "she said \"hi\" \\ ok"

    string whitespace = "a\tb\nc\rd";
    io:println(whitespace.toBalString()); // @output "a\tb\nc\rd"

    // control characters outside \t/\n/\r have no short escape in Ballerina
    // (unlike Go's \a/\b/\f/\v/\xHH), so they render as \u{hex}.
    string ctrl = "a\u{7}b\u{8}c\u{7f}d";
    io:println(ctrl.toBalString()); // @output "a\u{7}b\u{8}c\u{7f}d"

    map<anydata> ctrlKey = {};
    ctrlKey["a\u{8}b"] = 1;
    io:println(ctrlKey.toBalString()); // @output {"a\u{8}b":1}

    boolean b = true;
    io:println(b.toBalString()); // @output true

    float f1 = 12.342;
    io:println(f1.toBalString()); // @output 12.342

    float nanVal = 0.0 / 0.0;
    io:println(nanVal.toBalString()); // @output float:NaN

    float infVal = 1.0 / 0.0;
    io:println(infVal.toBalString()); // @output float:Infinity

    float negInfVal = -1.0 / 0.0;
    io:println(negInfVal.toBalString()); // @output float:-Infinity

    decimal d1 = 345.2425341;
    io:println(d1.toBalString()); // @output 345.2425341d

    decimal d2 = 1;
    io:println(d2.toBalString()); // @output 1d

    () nilVal = ();
    io:println(nilVal.toBalString()); // @output ()

    anydata[] data = [1, "Sam", 12.3, 12.12d, {value: 12}];
    io:println(data.toBalString()); // @output [1,"Sam",12.3,12.12d,{"value":12}]

    record {} key = {id: 5};
    io:println(key.toBalString()); // @output {"id":5}

    Address addr = {city: "Colombo", country: "Sri Lanka"};
    io:println(addr.toBalString()); // @output {"city":"Colombo","country":"Sri Lanka"}

    map<anydata> nested = {"a": 1, "b": {"c": 2}};
    io:println(nested.toBalString()); // @output {"a":1,"b":{"c":2}}

    int[] nums = [1, 2, 3];
    io:println(nums.toBalString()); // @output [1,2,3]

    // toBalString isn't callable directly on error (error isn't a subtype of
    // any), so nest it in a list to reach Error.BalString: detail values must
    // keep their expression-style forms (decimal "d" suffix, "()" for nil),
    // not the informal ones (no suffix, empty string).
    error e1 = error("boom", code = 42, ratio = 1.5d, note = ());
    (any|error)[] errWrapper = [e1];
    io:println(errWrapper.toBalString()); // @output [error("boom",code=42,ratio=1.5d,note=())]

    error cause = error("failure1");
    error e2 = error("failure2", cause, extra = 3.14d);
    (any|error)[] causeWrapper = [e2];
    io:println(causeWrapper.toBalString()); // @output [error("failure2",error("failure1"),extra=3.14d)]

    // A detail key that isn't a bare identifier (from a quoted identifier
    // named-arg) must still render as a valid identifier, quoted this time.
    error e3 = error("boom", 'a\-b = 5);
    (any|error)[] keyWrapper = [e3];
    io:println(keyWrapper.toBalString()); // @output [error("boom",'a\\\-b=5)]

    // a quoted identifier key that starts with a digit fails on the first
    // rune, a distinct case from 'a\-b above (which fails past the first).
    error e4 = error("boom", '1abc = 5);
    (any|error)[] digitKeyWrapper = [e4];
    io:println(digitKeyWrapper.toBalString()); // @output [error("boom",'1abc=5)]

    // cloneWithType failures carry a distinct error type name, which renders
    // as "error TypeName (...)" rather than the bare "error(...)".
    anydata badVal = "abc";
    int|error convErr = badVal.cloneWithType(int);
    (any|error)[] convErrWrapper = [convErr];
    io:println(convErrWrapper.toBalString());
    // @output [error {ballerina/lang.value}ConversionError ("'\"abc\"' value cannot be converted to 'int'",message="'\"abc\"' value cannot be converted to 'int'")]

    json jsonVal = {a: "STRING", b: 12, c: 12.4, d: true, e: {x: "x", y: ()}};
    io:println(jsonVal.toBalString()); // @output {"a":"STRING","b":12,"c":12.4,"d":true,"e":{"x":"x","y":()}}

    [string, int, decimal, float] tupleVal = ["TOM", 10, 90.12, 0.0 / 0.0];
    io:println(tupleVal.toBalString()); // @output ["TOM",10,90.12d,float:NaN]

    anydata[] selfRef = [1, 2];
    selfRef[2] = selfRef;
    io:println(selfRef.toBalString()); // @output [1,2,[...]]

    // a list that is its own first element is a distinct special case from
    // the general cycle above.
    anydata[] selfFirst = [];
    selfFirst.push(selfFirst);
    io:println(selfFirst.toBalString()); // @output [...]

    map<anydata> selfMap = {};
    selfMap["self"] = selfMap;
    io:println(selfMap.toBalString()); // @output {"self":{...}}

    // function/object/stream/typedesc/xml have no expression-style form of
    // their own, so they fall back to the informal renderer.
    function (int) returns int fn = x => x + 1;
    any fnVal = fn;
    io:println(fnVal.toBalString()); // @output function $anon/tobalstring-v:$anonFunc$_0

    Counter counter = new Counter();
    any objVal = counter;
    io:println(objVal.toBalString()); // @output object

    var strm = new stream<int, error?>(new Counter());
    any streamVal = strm;
    io:println(streamVal.toBalString()); // @output stream

    typedesc<anydata> td = int;
    any typedescVal = td;
    io:println(typedescVal.toBalString()); // @output typedesc

    xml xmlVal = xml `<a>hi</a>`;
    any xmlAny = xmlVal;
    io:println(xmlAny.toBalString()); // @output <a>hi</a>

    // future has no case at all (unlike the fallbacks above), so it reaches
    // BalString's default arm.
    future<int> fut = start compute();
    any futVal = fut;
    io:println(futVal.toBalString()); // @output <unsupported>
    int|error result = wait fut;
    io:println(result is int); // @output true
}

function compute() returns int {
    return 42;
}
