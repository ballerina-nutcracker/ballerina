// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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
    // fprint writes to stdout without appending a newline; calls accumulate on the same line
    io:fprint(io:stdout, "hello");
    io:fprint(io:stdout, " world");
    io:fprintln(io:stdout); // @output hello world

    // multiple values in a single fprint call
    io:fprint(io:stdout, "a", "b", "c");
    io:fprintln(io:stdout); // @output abc

    // fprint with integer values
    io:fprint(io:stdout, 1);
    io:fprint(io:stdout, 2);
    io:fprintln(io:stdout); // @output 12
}
