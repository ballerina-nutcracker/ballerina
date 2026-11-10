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

type R record {|
    float x;
|};

float modF = <float>(7 / 2);

public function main() {
    float f = <float>99999999999999999999;
    io:println(f); // @output 1e20
    io:println(<float>5); // @output 5.0
    io:println(<float>-99999999999999999999); // @output -1e20
    io:println(modF); // @output 3.5

    int|decimal x = <int|decimal>2.5;
    io:println(x); // @output 2.5
    io:println(x is decimal); // @output true
    any y = <int|decimal>2.5;
    io:println(y is decimal); // @output true
    any b = <float|decimal>5;
    io:println(b is float); // @output true
    any a = <int|float>(7 / 2);
    io:println(a); // @output 3

    io:println(<float>(7 / 2)); // @output 3.5
    io:println(<decimal>(1 / 3)); // @output 0.3333333333333333333333333333333333
    io:println(<float>(7 % 4)); // @output 3.0
    io:println(<float>(99999999999999999999 + 1)); // @output 1e20
    io:println(<decimal>(0.1 + 0.2)); // @output 0.3
    int i = 10;
    io:println(<float>(2 * i)); // @output 20.0
    io:println(<float>(i * 2)); // @output 20.0

    any fs = <float[]>[1, 2];
    io:println(fs); // @output [1.0,2.0]
    io:println(fs is float[]); // @output true
    any t = <[float, decimal]>[1, 2];
    io:println(t is [float, decimal]); // @output true
    any m = <map<float>>{a: 1};
    io:println(m); // @output {"a":1.0}
    io:println(m is map<float>); // @output true
    any r = <R>{x: 1};
    io:println(r is R); // @output true
    any lm = <map<decimal>[]>[{a: 1}];
    io:println(lm is map<decimal>[]); // @output true

    readonly ro = <readonly>[1, 2];
    io:println(ro is readonly); // @output true
    any ro2 = <float[] & readonly>[1, 2];
    io:println(ro2 is float[] & readonly); // @output true

    io:println(<float><any>99999999999999999999); // @output 1e20
    any n = <int|decimal><anydata>2.5;
    io:println(n is decimal); // @output true
}
