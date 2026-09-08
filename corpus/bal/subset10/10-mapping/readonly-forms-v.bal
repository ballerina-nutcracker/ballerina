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

public function main() {
    int age = 36;
    var explicitField = {readonly age: 36};
    io:println(explicitField); // @output {"age":36}
    var shorthand = {readonly age};
    io:println(shorthand); // @output {"age":36}
    var stringKey = {readonly "age": 36};
    io:println(stringKey); // @output {"age":36}

    map<int> contextualExplicit = {readonly age: 36};
    io:println(contextualExplicit); // @output {"age":36}
    map<int> contextualShorthand = {readonly age};
    io:println(contextualShorthand); // @output {"age":36}
    map<int> contextualStringKey = {readonly "age": 36};
    io:println(contextualStringKey); // @output {"age":36}
}
