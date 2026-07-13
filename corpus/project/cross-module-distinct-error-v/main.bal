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

import testorg/cross_module_distinct_error_v.other;
import testorg/cross_module_distinct_error_v.types;

public function main() {
    types:DistinctError distinctErr = types:createDistinctError(1);
    io:println(distinctErr is types:BaseError); // @output true
    io:println(distinctErr is other:OtherDistinctError); // @output false

    other:OtherDistinctError otherDistinctErr = other:createOtherDistinctError(1);
    io:println(otherDistinctErr is types:BaseError); // @output true
    io:println(otherDistinctErr is types:DistinctError); // @output false

    other:CombinedError combinedErr = other:createCombinedError(1);
    io:println(combinedErr is types:DistinctError); // @output true
    io:println(combinedErr is other:OtherDistinctError); // @output true

    types:BaseError baseErr = types:createBaseError(1);
    io:println(baseErr is types:DistinctError); // @output false
    io:println(baseErr is other:OtherDistinctError); // @output false
}
