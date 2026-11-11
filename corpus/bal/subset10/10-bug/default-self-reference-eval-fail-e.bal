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
isolated function runtimeValue() returns int { return 7; }
// runtime's nonconstant default would keep next from being folded, but next's
// evaluation fails before it reaches the omitted fields, as it would at runtime.
type R record {|
    R? next = (1 / 0 == 0) ? () : {}; // @error division by zero in a default that is never used
    int runtime = runtimeValue();
|};
public function main() {
    R r = {next: ()};
    io:println(r);
}
