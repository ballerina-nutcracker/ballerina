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

type Person record {|
    int deptId;
    string name;
|};

type Department record {|
    int id;
    int locationId;
|};

type Location record {|
    int id;
    string name;
|};

public function main() {
    Person[] people = [
        {deptId: 1, name: "Alex"}
    ];
    map<Department> departments = {
        hr: {id: 1, locationId: 10}
    };
    Location[] locations = [
        {id: 10, name: "HQ"}
    ];

    string[] _ = from var person in people
        join var dept in departments
        on person.deptId equals dept.id
        join var loc in locations
        on dept.locationId equals loc.name // @error
        select person.name;
}
