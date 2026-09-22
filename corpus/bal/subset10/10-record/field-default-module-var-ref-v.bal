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

type Config record {|
    int port = defaultPort;
    string host = defaultHost;
    int timeout = defaultPort + 1;
|};

final int defaultPort = 8080;
final string defaultHost = "localhost";

// A module-level variable initializer that itself uses the record keeps the
// module-init ordering exercised.
Config config = {host: "example.com"};

public function main() {
    Config c = {};
    io:println(c.port); // @output 8080
    io:println(c.host); // @output localhost
    io:println(c.timeout); // @output 8081
    io:println(config.host); // @output example.com
    io:println(config.port); // @output 8080
}
