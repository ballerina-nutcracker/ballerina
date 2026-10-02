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

function brokenBody() returns int {
    int bad = ; // @error
    int '_ = 1; // @error bad statement
    int| malformedType = 1; // @error
    int unknown = absent + 1; // @error
    int invocation = probe(absentArgument); // @error
    unknown = 2;
    var unavailable = absentInferred; // @error
    int dependent = unavailable + 1; // @error unused dependent
    { int nested = missing; probe("inner"); } // @error
    if (true && ) { probe("skippedBad"); } // @error
    if missingCondition { probe("skipped"); } // @error
    if true { int badInner = ; probe("innerIf"); } // @error
    while missingLoop { probe("skipped"); } // @error
    foreach var item in missingCollection { probe("skipped"); } // @error
    foreach Unknown item in [1] { probe("skipped"); } // @error
    match missingMatch { _ => { probe("skipped"); } } // @error
    match 1 { missingPattern => { probe("skipped"); } } // @error
    match 1 { _ if missingGuard => { probe("skipped"); } } // @error
    probe("outer"); // @error later outer sibling resolved
    int later = 4;
    return later;
}
function probe(int value) returns int => value;
function unavailableSignature(Unknown value) returns int { // @error
    probe("signatureSkipped");
    return 1;
}
class Survivor {
    function method() returns int {
        int bad = ; // @error
        return 5;
    }
}
function callers(Survivor survivor) returns int {
    int named = brokenBody();
    int method = survivor.method();
    return named + method;
}
