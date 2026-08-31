// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

function open(typedesc T = <>) returns T[] = external;
function fixed(typedesc T = <>) returns T[2] = external;
function zero(typedesc T = <>) returns T[0] = external;
function nested(typedesc T = <>) returns T[][] = external;
function outsideUnion(typedesc T = <>) returns T[]|error = external;
function insideUnion(typedesc<anydata> T = <>) returns (T|error)[] = external;
function insideIntersection(typedesc T = <>) returns (T & readonly)[] = external;
function constrained(typedesc<int|string> T = <>) returns T[] = external;
function fixedArraySibling(typedesc<string> T = <>) returns T[]|int[1] = external;

type StringArray string[];

public function main() {
    string[] _ = open();
    string[2] _ = fixed();
    string[0] _ = zero();
    string[][] _ = nested();
    string[]|error _ = outsideUnion();
    (string|error)[] _ = insideUnion();
    (string & readonly)[] _ = insideIntersection();
    string[][] _ = open(StringArray);
    string[]|int[1] _ = fixedArraySibling();

    string _ = open(); // @error
    string[3] _ = fixed(); // @error
    var _ = open(); // @error
    float[] _ = constrained(); // @error
}
