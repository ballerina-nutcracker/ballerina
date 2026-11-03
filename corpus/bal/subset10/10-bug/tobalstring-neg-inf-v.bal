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

public function main() {
    float negInf = -float:Infinity;
    float nan = float:NaN;
    io:println(negInf.toBalString()); // @output -float:Infinity
    io:println([negInf].toBalString()); // @output [-float:Infinity]

    map<float> m = {a: negInf};
    io:println(m.toBalString()); // @output {"a":-float:Infinity}

    anydata[] nested = [[negInf], {k: negInf}];
    io:println(nested.toBalString()); // @output [[-float:Infinity],{"k":-float:Infinity}]

    [float, string] tup = [negInf, "x"];
    io:println(tup.toBalString()); // @output [-float:Infinity,"x"]

    error e = error("boom", v = negInf);
    (any|error)[] errs = [e];
    io:println(errs.toBalString()); // @output [error("boom",v=-float:Infinity)]

    io:println((-negInf).toBalString()); // @output float:Infinity
    io:println((-nan).toBalString()); // @output float:NaN
    io:println((-1.5).toBalString()); // @output -1.5
    io:println(negInf.toString()); // @output -Infinity
}
