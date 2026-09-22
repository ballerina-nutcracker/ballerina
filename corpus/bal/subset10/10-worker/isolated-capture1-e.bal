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

isolated function readsNonFinal() returns int {
    int x = 1;
    worker w returns int {
        return x; // @error captured variable is not final
    }
    return wait w;
}

isolated function readsNonReadonly() returns int {
    final int[] arr = [1, 2];
    worker w returns int {
        return arr.length(); // @error captured variable is not readonly
    }
    return wait w;
}

isolated function assignsCaptured() returns int {
    int x = 1;
    worker w {
        x = 2; // @error captured variable is not final
    }
    wait w;
    return x;
}

public function main() {
    io:println(readsNonFinal());
    io:println(readsNonReadonly());
    io:println(assignsCaptured());
}
