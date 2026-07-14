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
    null a = null;
    null b = ();
    () c = null;
    () d = ();
    io:println(a == b); // @output true
    io:println(a == c); // @output true
    io:println(a == d); // @output true
    io:println(b == c); // @output true
    io:println(b == d); // @output true

    io:println(a == retNul1()); // @output true
    io:println(a == retNul2()); // @output true
    io:println(a == retNul3()); // @output true
    io:println(a == retNul4()); // @output true
    io:println(a == retNul5()); // @output true
}

function retNul1() {
    return;
}

function retNul2() returns () {
    return ();
}

function retNul3() returns () {
    return null;
}

function retNul4() returns null {
    return null;
}

function retNul5() returns null {
    return ();
}
