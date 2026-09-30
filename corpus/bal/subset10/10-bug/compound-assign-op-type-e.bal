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

byte g = 1;

type R record {|
    byte a;
|};

class C {
    byte b = 1;

    function inc() {
        self.b += 1; // @error byte + int is int
    }
}

function byteShift() {
    byte b = 200;
    b <<= 1; // @error byte << int is int
}

function byteAdd() {
    byte b = 200;
    b += 1; // @error byte + int is int
}

function byteMul() {
    byte b = 200;
    b *= 2; // @error byte * int is int
}

function byteOr() {
    byte b = 1;
    int i = 3;
    b |= i; // @error byte | int is int
}

function byteXor() {
    byte b = 1;
    int i = 3;
    b ^= i; // @error byte ^ int is int
}

function narrowedByteAdd() {
    byte|string v = 1;
    if v is byte {
        v += 1; // @error operand type is the declared byte|string
    }
}

function moduleByteAdd() {
    g += 1; // @error byte + int is int
}

function charConcat() {
    string:Char c = "a";
    c += "b"; // @error string:Char + string is string
}

function signed8Add() {
    int:Signed8 x = 1;
    x += 1; // @error int:Signed8 + int is int
}

function intMulFloat() {
    int i = 1;
    float f = 2.0;
    i *= f; // @error int * float is float
}

function byteIndexAdd() {
    byte[] bs = [1];
    bs[0] += 1; // @error byte + int is int
}

function byteFieldShift() {
    R r = {a: 1};
    r.a <<= 1; // @error byte << int is int
}

function byteObjectFieldAdd() {
    C c = new;
    c.b += 1; // @error byte + int is int
}

function byteTupleMemberAdd() {
    [byte, int] t = [1, 2];
    t[0] += 1; // @error byte + int is int
}

function capturedByteAdd() {
    byte x = 3;
    var f = function() {
        x += 1; // @error byte + int is int
    };
    f();
}

function singletonAdd() {
    1|2 v = 1;
    v += 1; // @error 1|2 + int is int
}

function nilableRhs() {
    int x = 1;
    int? y = 2;
    x += y; // @error int + int? is int?, not a subtype of int
}

public function main() {
    byteShift();
    byteAdd();
    byteMul();
    byteOr();
    byteXor();
    narrowedByteAdd();
    moduleByteAdd();
    charConcat();
    signed8Add();
    intMulFloat();
    byteIndexAdd();
    byteFieldShift();
    byteObjectFieldAdd();
    byteTupleMemberAdd();
    capturedByteAdd();
    singletonAdd();
    nilableRhs();
}

function narrowedUnion() {
    int|string v = 1;
    if v is int {
        v += 1; // @error operand type is the declared int|string
    }
}

function narrowedNilable() {
    int? n = 1;
    if n is int {
        n += 1; // @error operand type is the declared int?
    }
}
