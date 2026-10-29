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

isolated int counter = 0;

isolated function copy(int[] xs) returns int[] {
    return [xs[0]];
}

function startInLockWithLockLocal() {
    lock {
        counter += 1;
        int[] local = [counter];
        future<int[]> f = start copy(local); // @error start action inside lock
        _ = f;
    }
}

function startInLockWithOutsideLocal() {
    int[] outside = [1];
    lock {
        counter += 1;
        future<int[]> f = start copy(outside); // @error start action inside lock
        _ = f;
    }
}

function startInLockWithQueryLocal() {
    lock {
        counter += 1;
        future<int[]> f = start copy(from int x in [counter] select x); // @error start action inside lock
        _ = f;
    }
}

public function main() {
}
