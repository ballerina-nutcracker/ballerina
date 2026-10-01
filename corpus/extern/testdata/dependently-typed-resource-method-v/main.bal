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

import dependentresourcemethod.api;
import ballerina/io;

public function main() {
    api:Client c = new ();

    string explicit = checkpanic c->/values/one/two(string);
    io:println(explicit); // @output explicit

    string seg = "one";
    string computed = checkpanic c->/values/[seg](string);
    io:println(computed); // @output explicit

    int computedInferred = checkpanic c->/values/[seg]();
    io:println(computedInferred); // @output 42

    string mixed = checkpanic c->/values/one/[seg](string);
    io:println(mixed); // @output explicit

    int multiComputed = checkpanic c->/values/[seg]/[seg + "x"]();
    io:println(multiComputed); // @output 42

    string literalSeg = checkpanic c->/values/["lit"](string);
    io:println(literalSeg); // @output explicit

    int inferred = checkpanic c->/values/three();
    io:println(inferred); // @output 42

    api:ItemClient items = new ();
    int id = 7;
    string item = checkpanic items->/items/[id](string);
    io:println(item); // @output item 7

    int doubled = checkpanic items->/items/[id + 1]();
    io:println(doubled); // @output 16
}
