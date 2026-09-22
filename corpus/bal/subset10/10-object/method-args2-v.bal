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

type Value object {
    public function get(int value, int increment = 2) returns int;
};

readonly class ReadonlyValue {
    public function get(int value, int increment = 2) returns int {
        return value + increment;
    }
}

type Caller client object {
    remote function add(int value, int increment = 2) returns int;
};

client class CallerImpl {
    remote function add(int value, int increment = 2) returns int {
        return value + increment;
    }
}

function useReadonly(readonly & Value value) returns int {
		return value.get(value = 3);
}

public function main() {
	Value value = new ReadonlyValue();

	object {}|int narrowed = value;
    if narrowed is object {
        public function get(int value, int increment = 2) returns int;
    } {
        io:println(narrowed.get(5, 4)); // @output 9
    }

    Caller caller = new CallerImpl();
	int result = caller->add(value = 6);
	io:println(result); // @output 8
}
