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

public type Values record {|
    int left = 5;
    int right = 10;
|};

public class DefaultInit {
    public int value;

    public function init(int value = 10) {
        self.value = value;
    }
}

public class NamedInit {
    public int value;

    public function init(int left, int right) {
        self.value = left + right;
    }
}

public class IncludedInit {
    public int value;

    public function init(*Values values) {
        self.value = values.left + values.right;
    }
}

public type Calculator object {
    public function withDefaults(int left = 5, int right = 10) returns int;
    public function withNamed(int left, int right) returns int;
    public function withIncluded(int base, *Values values) returns int;
};

public class CalculatorImpl {
    *Calculator;

    public function withDefaults(int left = 5, int right = 10) returns int {
        return left + right;
    }

    public function withNamed(int left, int right) returns int {
        return left + right;
    }

    public function withIncluded(int base, *Values values) returns int {
        return base + values.left + values.right;
    }
}
