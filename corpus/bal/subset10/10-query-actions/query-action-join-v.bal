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

type Employee record {|
    string name;
    string dept;
    int age;
|};

type Department record {|
    string code;
    string title;
|};

public function main() {
    Employee[] employees = [
        {name: "Alex", dept: "OPS", age: 23},
        {name: "Ranjan", dept: "ENG", age: 30},
        {name: "Mia", dept: "ENG", age: 27},
        {name: "John", dept: "HR", age: 33}
    ];
    Department[] departments = [
        {code: "ENG", title: "Engineering"},
        {code: "OPS", title: "Operations"},
        {code: "HR", title: "People"}
    ];

    string labels = "";
    int ageTotal = 0;
    error? result = from var employee in employees
        join var department in departments
        on employee.dept equals department.code
        where department.title == "Engineering"
        let int nextAge = employee.age + 1
        order by employee.age descending
        limit 2
        do {
            labels = labels + employee.name + ":" + department.title + ";";
            ageTotal += nextAge;
        };

    io:println(result is ()); // @output true
    io:println(labels); // @output Ranjan:Engineering;Mia:Engineering;
    io:println(ageTotal); // @output 59
}
