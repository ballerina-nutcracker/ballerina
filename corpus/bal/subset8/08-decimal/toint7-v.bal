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
    decimal d = 1d;
    io:println(<int>d); // @output 1

    d = 0d;
    io:println(<int>d); // @output 0

    d = -1d;
    io:println(<int>d); // @output -1

    d = 0.5d;
    io:println(<int>d); // @output 0

    d = 1.5d;
    io:println(<int>d); // @output 2

    d = 2.5d;
    io:println(<int>d); // @output 2

    d = -0.5d;
    io:println(<int>d); // @output 0

    d = -1.5d;
    io:println(<int>d); // @output -2

    d = -2.5d;
    io:println(<int>d); // @output -2

    d = 1e2d;
    io:println(<int>d); // @output 100

    d = 1.5e2d;
    io:println(<int>d); // @output 150

    d = -1.5e2d;
    io:println(<int>d); // @output -150

    d = 1.51e2d;
    io:println(<int>d); // @output 151

    d = 1.513e2d;
    io:println(<int>d); // @output 151

    d = 1.515e2d;
    io:println(<int>d); // @output 152

    d = 9223372036854775807d;
    io:println(<int>d); // @output 9223372036854775807

    d = 9223372036854775807.0d;
    io:println(<int>d); // @output 9223372036854775807

    d = 9223372036854775807.1d;
    io:println(<int>d); // @output 9223372036854775807

    d = 9223372036854775807.4d;
    io:println(<int>d); // @output 9223372036854775807

    d = 92233720368547758074e-1d;
    io:println(<int>d); // @output 9223372036854775807

    d = 1E-6143d;
    io:println(<int>d); // @output 0

    d = -1E-6143d;
    io:println(<int>d); // @output 0
}
