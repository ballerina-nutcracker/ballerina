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

type Operations record {|
    int unaryPlus;
    float negFloat;
    decimal negDecimal;
    boolean notFalse;
    boolean shortAnd;
    boolean shortOr;
    int addInt;
    float addFloat;
    decimal addDecimal;
    int subInt;
    float subFloat;
    decimal subDecimal;
    int mulInt;
    float mulFloat;
    decimal mulDecimal;
    float mulFloatInt;
    float mulIntFloat;
    decimal mulDecimalInt;
    decimal mulIntDecimal;
    int divInt;
    float divFloat;
    decimal divDecimal;
    float divFloatInt;
    decimal divDecimalInt;
    int modInt;
    float modFloat;
    decimal modDecimal;
    boolean equal;
    boolean notEqual;
    boolean refEqual;
    boolean refNotEqual;
    boolean nilRefEqual;
    boolean floatRefEqual;
    boolean stringRefEqual;
    boolean booleanRefEqual;
    boolean decimalRefEqual;
    boolean listRefEqual;
    boolean mapRefEqual;
    boolean greater;
    boolean greaterEqual;
    boolean less;
    boolean lessEqual;
    int bitAnd;
    int bitOr;
    int bitXor;
    int leftShift;
    int rightShift;
    int unsignedRightShift;
    int floatToInt;
    int decimalToInt;
    float intToFloat;
    float decimalToFloat;
    decimal intToDecimal;
    decimal floatToDecimal;
    int identityConversion;
    int? nilUnary;
    int? nilMultiply;
    boolean? nilLess;
|};

annotation Operations operations on type;

@operations {
    unaryPlus: +5,
    negFloat: -2.5,
    negDecimal: -2.5d,
    notFalse: !false,
    shortAnd: false && true,
    shortOr: true || false,
    addInt: 20 + 2,
    addFloat: 1.5 + 2.5,
    addDecimal: 1.5d + 2.5d,
    subInt: 20 - 2,
    subFloat: 5.5 - 2.5,
    subDecimal: 5.5d - 2.5d,
    mulInt: 6 * 7,
    mulFloat: 2.5 * 2.0,
    mulDecimal: 2.5d * 2d,
    mulFloatInt: 2.5 * 2,
    mulIntFloat: 2 * 2.5,
    mulDecimalInt: 2.5d * 2,
    mulIntDecimal: 2 * 2.5d,
    divInt: 21 / 3,
    divFloat: 7.5 / 2.5,
    divDecimal: 7.5d / 2.5d,
    divFloatInt: 7.5 / 3,
    divDecimalInt: 7.5d / 3,
    modInt: 22 % 5,
    modFloat: 7.5 % 2.0,
    modDecimal: 7.5d % 2d,
    equal: 2 == 2,
    notEqual: 2 != 2,
    refEqual: 2 === 2,
    refNotEqual: 2 !== 2,
    nilRefEqual: () === (),
    floatRefEqual: 2.5 === 2.5,
    stringRefEqual: "value" === "value",
    booleanRefEqual: true === true,
    decimalRefEqual: 2.5d === 2.5d,
    listRefEqual: [1, 2] === [1, 2],
    mapRefEqual: {value: 1} === {value: 1},
    greater: 3 > 2,
    greaterEqual: 3 >= 3,
    less: 2 < 3,
    lessEqual: 3 <= 3,
    bitAnd: 7 & 3,
    bitOr: 4 | 3,
    bitXor: 7 ^ 3,
    leftShift: 3 << 2,
    rightShift: -8 >> 2,
    unsignedRightShift: -1 >>> 63,
    floatToInt: <int>2.5,
    decimalToInt: <int>2.5d,
    intToFloat: <float>2,
    decimalToFloat: <float>2.5d,
    intToDecimal: <decimal>2,
    floatToDecimal: <decimal>2.5,
    identityConversion: <int>2,
    nilUnary: -(<int?>()),
    nilMultiply: (<int?>()) * 2,
    nilLess: (<int?>()) < 2
}
type Target int;

public function main() {
    Operations? value = Target.@operations;
    if value is Operations {
        io:println(value.addInt); // @output 22
        io:println(value.mulDecimal); // @output 5.0
        io:println(value.unsignedRightShift); // @output 1
        io:println(value.floatToDecimal); // @output 2.5
    }
}
