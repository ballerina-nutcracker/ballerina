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

isolated class Box {
    private int n;

    isolated function init() {
        self.n = 3;
    }

    isolated function get() returns int {
        lock {
            return self.n;
        }
    }

    isolated function getInWorker() returns int {
        worker w returns int {
            lock {
                return self.n;
            }
        }
        return wait w;
    }
}

isolated function capturesIsolated() returns int {
    final readonly & int[] arr = [2];
    final Box box = new;
    final isolated function () returns int one = isolated function () returns int {
        return 1;
    };
    worker w returns int {
        return arr[0] + box.getInWorker() + one();
    }
    return wait w;
}

isolated function capturesParam(int p) returns int {
    worker w returns int {
        return p;
    }
    return wait w;
}

public function main() {
    io:println(capturesIsolated()); // @output 6
    io:println(capturesParam(4)); // @output 4
}
