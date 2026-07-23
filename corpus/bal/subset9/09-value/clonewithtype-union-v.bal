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

type IntOrString int|string;
type OptFloat float?;
type DecimalOrString decimal|string;
type IntNilable int?;
type IntStrArr int[]|string[];
type ByteOrString byte|string;

type PersonA record {|
    string name;
    int age;
|};

type PersonB record {|
    string name;
    string role;
|};

type PersonUnion PersonA|PersonB;

public function main() returns error? {
    // nil → nilable type: succeeds
    anydata nilVal = ();
    IntNilable n = check nilVal.cloneWithType(IntNilable);
    io:println(n is ()); // @output true

    // nil → non-nilable union: error
    io:println(nilVal.cloneWithType(IntOrString) is error); // @output true

    // union: first member (int) matches exactly
    anydata num = 42;
    IntOrString asInt = check num.cloneWithType(IntOrString);
    io:println(asInt); // @output 42

    // union: second member (string) matches exactly
    anydata str = "hello";
    IntOrString asStr = check str.cloneWithType(IntOrString);
    io:println(asStr); // @output hello

    // numeric coercion in union: float 2.0 doesn't match int or string exactly,
    // but numeric conversion float→int succeeds on the second pass
    anydata fv = 2.0;
    IntOrString coerced = check fv.cloneWithType(IntOrString);
    io:println(coerced); // @output 2

    // numeric coercion: int 3 → float? (second pass: int→float)
    anydata iv = 3;
    OptFloat optF = check iv.cloneWithType(OptFloat);
    io:println(optF); // @output 3.0

    // numeric coercion: int 10 → decimal|string (second pass: int→decimal)
    anydata dv = 10;
    DecimalOrString ds = check dv.cloneWithType(DecimalOrString);
    io:println(ds); // @output 10

    // struct union: first member (PersonA) matches
    anydata pMap = {name: "Alice", age: 30};
    PersonUnion pu = check pMap.cloneWithType(PersonUnion);
    io:println(pu); // @output {"name":"Alice","age":30}

    // struct union: PersonA fails (missing age), PersonB succeeds
    anydata pMap2 = {name: "Bob", role: "admin"};
    PersonUnion pu2 = check pMap2.cloneWithType(PersonUnion);
    io:println(pu2); // @output {"name":"Bob","role":"admin"}

    // array union: int[] matches first member
    anydata intArr = [1, 2, 3];
    IntStrArr arrVal = check intArr.cloneWithType(IntStrArr);
    io:println(arrVal); // @output [1,2,3]

    // array union: int[] fails for string[], string[] succeeds
    anydata strArr = ["a", "b"];
    IntStrArr strArrVal = check strArr.cloneWithType(IntStrArr);
    io:println(strArrVal); // @output ["a","b"]

    // numeric coercion in union: float→byte succeeds on second pass
    anydata bv = 200.0;
    ByteOrString asByte = check bv.cloneWithType(ByteOrString);
    io:println(asByte); // @output 200

    return;
}
