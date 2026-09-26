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

type Person record {|
    string firstName;
    string lastName;
    int age;
|};

public function main() {
    Person[] people = [
        {firstName: "Alex", lastName: "George", age: 23},
        {firstName: "Ranjan", lastName: "Fonseka", age: 30},
        {firstName: "Mia", lastName: "Silva", age: 19}
    ];

    string[] names = [];
    int totalAge = 0;
    error? result = from var person in people
        where person.age < 30
        let string fullName = person.firstName + " " + person.lastName
        do {
            names[names.length()] = fullName;
            totalAge += person.age;
        };

    io:println(result is ()); // @output true
    io:println(names.length()); // @output 2
    io:println(names[0]); // @output Alex George
    io:println(names[1]); // @output Mia Silva
    io:println(totalAge); // @output 42

    int nestedTotal = 0;
    result = from var n in [1, 2, 3]
        do {
            from var m in [n, n * 10]
                do {
                    nestedTotal += m;
                };
        };

    io:println(result is ()); // @output true
    io:println(nestedTotal); // @output 66

    map<boolean> flags = {enabled: true, archived: false, visible: true};
    int enabledCount = 0;
    result = from var flag in flags
        where flag
        do {
            enabledCount += 1;
        };

    io:println(result is ()); // @output true
    io:println(enabledCount); // @output 2
}
