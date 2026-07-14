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
    io:println(<int>1d); // @output 1
    io:println(<int>0d); // @output 0
    io:println(<int>-1d); // @output -1
    io:println(<int>0.5d); // @output 0
    io:println(<int>1.5d); // @output 2
    io:println(<int>2.5d); // @output 2
    io:println(<int>-0.5d); // @output 0
    io:println(<int>-1.5d); // @output -2
    io:println(<int>-2.5d); // @output -2
    io:println(<int>1e2d); // @output 100
    io:println(<int>1.5e2d); // @output 150
    io:println(<int>-1.5e2d); // @output -150
    io:println(<int>1.51e2d); // @output 151
    io:println(<int>1.513e2d); // @output 151
    io:println(<int>1.515e2d); // @output 152
    io:println(<int>9223372036854775807d); // @output 9223372036854775807
    io:println(<int>9223372036854775807.0d); // @output 9223372036854775807
    io:println(<int>9223372036854775807.1d); // @output 9223372036854775807
    io:println(<int>9223372036854775807.4d); // @output 9223372036854775807
    io:println(<int>92233720368547758074e-1d); // @output 9223372036854775807
    io:println(<int>1E-6143d); // @output 0
    io:println(<int>-1E-6143d); // @output 0
}
