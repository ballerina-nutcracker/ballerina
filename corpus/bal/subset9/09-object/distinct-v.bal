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

class Person {
    public string name;

    function init(string name) {
        self.name = name;
    }
}

// The `DistinctPerson` type is a proper subtype of the `Person` type.
distinct class DistinctPerson {
    public string name;

    function init(string name) {
        self.name = name;
    }
}

// The `SomeWhatDistinctPerson` type is a subtype of the `DistinctPerson` type
// since it includes the `DistinctPerson` type's type IDs via inclusion.
class SomeWhatDistinctPerson {
    *DistinctPerson;

    public string name;

    function init(string name) {
        self.name = name;
    }
}

// The `EvenMoreDistinctPerson` type is a proper subtype of the `DistinctPerson`
// type since it has an additional type ID.
distinct class EvenMoreDistinctPerson {
    *DistinctPerson;

    public string name;

    function init(string name) {
        self.name = name;
    }
}

public function main() {
    Person person = new ("John Smith");
    io:println(person is DistinctPerson); // @output false

    DistinctPerson distinctPerson = new ("Alice Johnson");
    io:println(distinctPerson is Person); // @output true

    SomeWhatDistinctPerson someWhatDistinctPerson = new ("Michael Brown");
    io:println(someWhatDistinctPerson is DistinctPerson); // @output true
    io:println(distinctPerson is SomeWhatDistinctPerson); // @output true

    EvenMoreDistinctPerson evenMoreDistinctPerson = new ("Sarah Wilson");
    io:println(evenMoreDistinctPerson is DistinctPerson); // @output true
    io:println(distinctPerson is EvenMoreDistinctPerson); // @output false
}
