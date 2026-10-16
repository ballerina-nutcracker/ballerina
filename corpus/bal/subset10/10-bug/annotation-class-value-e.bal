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

type Info record {|
    int value;
|};

type Flag record {|
    boolean value;
|};

annotation Info classInfo on class;
annotation Flag classFlag on class;
annotation Info fieldInfo on object field;
annotation Flag methodFlag on object function;
annotation Info returnInfo on return;
annotation Info serviceInfo on service;
annotation Flag serviceFlag on service;

@classInfo {value: "str"} // @error
class Counter {
    @fieldInfo {value: true} // @error
    int count = 0;

    @methodFlag {value: 1 == "a"} // @error
    function get() returns @returnInfo {value: "str"} int { // @error
        return self.count;
    }
}

@classFlag {value: 1 == "a"} // @error
class Other {
    @methodFlag // @error
    function reset() {
    }
}

@serviceInfo {value: "str"} // @error
service class Service {
}

@serviceFlag {value: 1 == "a"} // @error
service class OtherService {
}

public function main() {
}
