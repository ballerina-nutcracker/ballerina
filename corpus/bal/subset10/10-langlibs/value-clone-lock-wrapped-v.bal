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

isolated int[] data = [1, 2];

isolated function viaCast() returns int[] {
    lock {
        return <int[]>data.clone();
    }
}

isolated function viaTrap() returns int[]|error {
    lock {
        return trap data.clone();
    }
}

isolated function viaCheck() returns int[]|error {
    lock {
        return check trap data.clone();
    }
}

public function main() returns error? {
    int[] a = viaCast();
    a.push(3);
    int[] b = check viaTrap();
    b.push(4);
    int[] c = check viaCheck();
    c.push(5);
    io:println(a); // @output [1,2,3]
    io:println(b); // @output [1,2,4]
    io:println(c); // @output [1,2,5]
    io:println(viaCast()); // @output [1,2]
}
