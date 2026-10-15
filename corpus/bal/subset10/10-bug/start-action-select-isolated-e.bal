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


function one(int i) returns int {
    return i;
}

isolated function isolatedOne(int i) returns int {
    return i;
}

isolated int[] values = [];

isolated function startNonIsolated() returns future<int>[] {
    return from int i in [1, 2] select start one(i); // @error start action is not isolated
}

isolated function startInLock() {
    lock {
        future<int>[] started = from int i in [1, 2] select start isolatedOne(i); // @error cannot start strand inside a lock statement
        values.push(started.length());
    }
}

public function main() {
    _ = startNonIsolated();
    startInLock();
}
