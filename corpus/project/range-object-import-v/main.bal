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
import testorg/range_object_import_v.util;

// A non-root module that explicitly imports lang.object does not change the
// order in which the bundled lang libraries are compiled: lang.__internal,
// which lang.object imports, is still compiled before it.
public function main() {
    io:println(util:isIterable(1 ..< 3)); // @output true
    io:println(util:isIterable(1)); // @output false
}
