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

import testorg/cross_module_distinct_object_v.types;

public function main() {
    types:Person person = new ("John Smith");
    io:println(person is types:DistinctPerson); // @output false

    types:DistinctPerson distinctPerson = new ("Alice Johnson");
    io:println(distinctPerson is types:Person); // @output true

    types:SomeWhatDistinctPerson someWhatDistinctPerson = new ("Michael Brown");
    io:println(someWhatDistinctPerson is types:DistinctPerson); // @output true
    io:println(distinctPerson is types:SomeWhatDistinctPerson); // @output true

    types:EvenMoreDistinctPerson evenMoreDistinctPerson = new ("Sarah Wilson");
    io:println(evenMoreDistinctPerson is types:DistinctPerson); // @output true
    io:println(distinctPerson is types:EvenMoreDistinctPerson); // @output false
}
