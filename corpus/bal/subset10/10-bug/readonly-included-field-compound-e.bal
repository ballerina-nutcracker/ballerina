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

class Counter {
    public int count = 7;
}

type CounterView object {
    *Counter;
};

readonly class ReadonlyCounter {
    *CounterView;

    function init() {
        self.count = 1;
        self.count += 1; // @error compound assignment to implicitly final included field
    }
}

class MutableCounter {
    *Counter;

    function init() {
        self.count = 1;
        self.count = 2;
        self.count += 1;
    }

    function increment() {
        self.count += 1;
    }
}

public function main() {
    ReadonlyCounter _ = new;
    MutableCounter mutableCounter = new;
    mutableCounter.increment();
}
