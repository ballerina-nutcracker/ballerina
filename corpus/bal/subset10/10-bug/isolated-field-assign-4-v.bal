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

isolated class Summary {
    private int total;
    private final int first;

    isolated function init(int[] arr, map<int> m = {a: 1}, int[] extra = [7, 8]) {
        int sum = 0;
        foreach int v in arr {
            sum += v;
        }
        if extra.length() > 0 {
            sum += extra[0];
        }
        self.total = sum + m.length();
        self.first = arr[0];
    }

    isolated function get() returns int {
        lock {
            return self.total + self.first;
        }
    }
}

isolated class Isolatable {
    private final int[] & readonly frozen;
    private int[] fresh;
    private final Summary summary;

    isolated function init(int[] & readonly frozen, int a, Summary summary) {
        self.frozen = frozen;
        self.fresh = [a, a + 1];
        self.summary = summary;
    }

    isolated function get() returns int {
        lock {
            return self.frozen[0] + self.fresh[1];
        }
    }

    isolated function summaryTotal() returns int {
        lock {
            return self.summary.get();
        }
    }
}

isolated class Chosen {
    private int[] arr;

    isolated function init(boolean b, int k) {
        self.arr = b ? [k] : [];
    }

    isolated function size() returns int {
        lock {
            return self.arr.length();
        }
    }
}

class Plain {
    int[] arr;

    function init(int[] arr) {
        self.arr = arr;
    }
}

public function main() {
    int[] shared = [1, 2, 3];
    Summary s = new(shared);
    shared[0] = 99;
    io:println(s.get()); // @output 15
    Summary s2 = new(shared, {}, []);
    io:println(s2.get()); // @output 203
    Isolatable i = new([10], 20, s);
    io:println(i.get()); // @output 31
    io:println(i.summaryTotal()); // @output 15
    Chosen c = new(true, 4);
    io:println(c.size()); // @output 1
    Plain p = new(shared);
    shared[1] = 5;
    io:println(p.arr[1]); // @output 5
}
