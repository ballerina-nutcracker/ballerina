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
    io:println(1d === 1d); // @output true
    io:println(0d === 0d); // @output true
    io:println(-9.999999999999999999999999999999999E6144d === -9.999999999999999999999999999999999E6144d); // @output true
    io:println(0.000001d === 0.000001d); // @output true
    io:println(1.0d === 1.00d); // @output false
    io:println(0.0d === 0d); // @output false
    io:println(0.00000100d === 0.000001d); // @output false
    io:println(1E-6142d === 10E-6143d); // @output false
    io:println(100E3d !== 10E4d); // @output true
}
