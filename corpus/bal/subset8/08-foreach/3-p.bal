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

class FailingIterator {
    int idx = 0;

    public function next() returns record {|int value;|}|error? {
        if self.idx >= 2 {
            return error("iterator failed");
        }
        int val = self.idx;
        self.idx += 1;
        return {value: val};
    }
}

class FailingIterable {
    *object:Iterable;

    public function iterator() returns FailingIterator {
        return new;
    }
}

public function main() {
    FailingIterable f = new;
    foreach int val in f { // @panic iterator failed
        io:println(val); // @output 0
                         // @output 1
    }
}
