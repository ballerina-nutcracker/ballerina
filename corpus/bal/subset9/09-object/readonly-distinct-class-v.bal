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

readonly distinct class Person {
    public string name;

    function init(string name) {
        self.name = name;
    }
}

public readonly distinct class Employee {
    *Person;

    public string name;

    function init(string name) {
        self.name = name;
    }
}

public function main() {
    Person person = new ("John Smith");
    io:println(person is Employee); // @output false

    Employee employee = new ("Alice Johnson");
    io:println(employee is Person); // @output true

    Person employeeAsPerson = employee;
    io:println(employeeAsPerson is Employee); // @output true
}
