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
import testorg/inline_record_default_cross_module.types;

public function main() {
    io:println(types:parameterDefault({})); // @output 41
    io:println(types:parameterDefault({parameterValue: 11})); // @output 11

    types:NestedAlias nested = {nested: {}};
    io:println(nested.nested.nestedValue); // @output 42
    types:NestedAlias nestedOverride = {nested: {nestedValue: 12}};
    io:println(nestedOverride.nested.nestedValue); // @output 12

    types:AliasList values = [{}];
    io:println(values[0].listValue); // @output 43

    io:println(types:restDefaults({}, {restValue: 10})); // @output 54

    types:Factory factory = () => {};
    io:println(types:invokeFactory(factory)); // @output 45
}
