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

type FloatArray float[];
type DecimalArray decimal[];
type IntArray int[];
type ByteArray byte[];
type IntMap map<int>;

type IntRec record {|
    int a;
    string b;
    int c;
|};

type FloatRec record {|
    float a;
    string b;
    float c;
|};

public function main() returns error? {
    // int → float
    anydata i = 1234;
    float f = check i.cloneWithType(float);
    io:println(f); // @output 1234.0

    // float → int: round-to-even (1234.6 → 1235)
    anydata fv = 1234.6;
    int iv = check fv.cloneWithType(int);
    io:println(iv); // @output 1235

    // float → int: round-to-even ties go to nearest even (2.5 → 2)
    anydata half = 2.5;
    int halfInt = check half.cloneWithType(int);
    io:println(halfInt); // @output 2

    // int → decimal
    anydata di = 7;
    decimal d = check di.cloneWithType(decimal);
    io:println(d); // @output 7

    // float → decimal
    anydata fdv = 1.5;
    decimal fd = check fdv.cloneWithType(decimal);
    io:println(fd); // @output 1.5

    // decimal → int (rounds to nearest: 12.3456 → 12)
    decimal decVal = 12.3456;
    anydata decAny = decVal;
    int fromDec = check decAny.cloneWithType(int);
    io:println(fromDec); // @output 12

    // decimal → float
    decimal decF = 3.5;
    anydata decFAny = decF;
    float floatFromDec = check decFAny.cloneWithType(float);
    io:println(floatFromDec); // @output 3.5

    // int → byte (0–255)
    anydata bv = 200;
    byte b = check bv.cloneWithType(byte);
    io:println(b); // @output 200

    // float → byte
    anydata fbv = 200.0;
    byte fb = check fbv.cloneWithType(byte);
    io:println(fb); // @output 200

    // decimal → byte (rounds to nearest: 42.9 → 43)
    decimal decByte = 42.9;
    anydata decByteAny = decByte;
    byte dby = check decByteAny.cloneWithType(byte);
    io:println(dby); // @output 43

    // int[] → float[]
    anydata ints = [1, 2, 3];
    FloatArray fa = check ints.cloneWithType(FloatArray);
    io:println(fa); // @output [1.0,2.0,3.0]

    // int[] → decimal[]
    DecimalArray da = check ints.cloneWithType(DecimalArray);
    io:println(da); // @output [1,2,3]

    // float[] → int[]
    anydata floats = [1.0, 2.0, 3.0];
    IntArray ia = check floats.cloneWithType(IntArray);
    io:println(ia); // @output [1,2,3]

    // float[] → byte[]
    anydata floatBytes = [1.0, 200.0, 255.0];
    ByteArray ba = check floatBytes.cloneWithType(ByteArray);
    io:println(ba); // @output [1,200,255]

    // decimal[] → int[]: recursive round-to-even, not truncation (1.6 → 2; ties
    // to even in both directions: 2.5 → 2, 3.5 → 4)
    decimal[] decArr = [1.6, 2.5, 3.5];
    anydata decArrAny = decArr;
    IntArray fromDecArr = check decArrAny.cloneWithType(IntArray);
    io:println(fromDecArr); // @output [2,2,4]

    // map<float> → map<int>: round-to-even (1.2 → 1, 2.7 → 3)
    anydata mf = {a: 1.2, b: 2.7};
    IntMap mi = check mf.cloneWithType(IntMap);
    io:println(mi); // @output {"a":1,"b":3}

    // record with int fields → record with float fields
    anydata r = {a: 21, b: "Alice", c: 1000};
    FloatRec fr = check r.cloneWithType(FloatRec);
    io:println(fr); // @output {"a":21.0,"b":"Alice","c":1000.0}

    // --- error cases ---
    anydata over = 256;
    io:println(over.cloneWithType(byte) is error); // @output true

    anydata neg = -1;
    io:println(neg.cloneWithType(byte) is error); // @output true

    float nan = 0.0/0.0;
    anydata nanAny = nan;
    io:println(nanAny.cloneWithType(int) is error); // @output true

    float inf = 1.0/0.0;
    anydata infAny = inf;
    io:println(infAny.cloneWithType(int) is error); // @output true

    return;
}
