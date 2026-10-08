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

int m = 1;
isolated int[] iv = [];

isolated class Holder {
    private int[] f = [1, 2];

    function discard() {
        lock {
            _ = self.f;
        }
    }
}

isolated function discardLength() {
    lock {
        _ = iv.length();
    }
}

public function main() {
    lock {
        _ = 1;
        _ = m;
        _ = iv;
        int x = 1;
        _ = x;
        iv.push(x);
    }
    discardLength();
    new Holder().discard();
    lock {
        io:println(iv.length()); // @output 1
    }
}
