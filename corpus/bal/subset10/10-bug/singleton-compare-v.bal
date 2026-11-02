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

const int MIN = 1;
const int MAX = 10;
const string NAME = "a";

public function main() {
    true ordered = MIN < MAX;
    io:println(ordered); // @output true

    true same = MIN == 1;
    io:println(same); // @output true

    false lt = 2.0 < 1.0;
    io:println(lt); // @output false

    string label = MIN <= MAX ? "ordered" : 0;
    io:println(label); // @output ordered

    true ge = MAX >= 10;
    io:println(ge); // @output true

    false gt = 1d > 2d;
    io:println(gt); // @output false

    false ne = NAME != "a";
    io:println(ne); // @output false

    false negated = !(MIN < MAX);
    io:println(negated); // @output false

    true zero = 0.0 == -0.0;
    io:println(zero); // @output true
    false negZeroLt = -0.0 < 0.0;
    io:println(negZeroLt); // @output false
    true negZeroLe = -0.0 <= 0.0;
    io:println(negZeroLe); // @output true

    true decEq = 1.0d == 1.00d;
    io:println(decEq); // @output true
    false decLt = 1.0d < 1.00d;
    io:println(decLt); // @output false

    true nilEq = () == ();
    io:println(nilEq); // @output true
    true nilLe = () <= ();
    io:println(nilLe); // @output true
    false nilLt = () < ();
    io:println(nilLt); // @output false
    false nilInt = () < 1;
    io:println(nilInt); // @output false
    false intNil = 1 >= ();
    io:println(intNil); // @output false

    false strCase = "a" < "B";
    io:println(strCase); // @output false
    true strPrefix = "ab" > "a";
    io:println(strPrefix); // @output true
    true strEmpty = "" < "a";
    io:println(strEmpty); // @output true

    true boolOrder = false < true;
    io:println(boolOrder); // @output true
    false boolLe = true <= false;
    io:println(boolLe); // @output false
}
