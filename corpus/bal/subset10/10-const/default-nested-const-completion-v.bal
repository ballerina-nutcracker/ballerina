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

// Inner's default reads the constant CONFIG while Outer's defaults are being
// resolved. CONFIG leaves out port, so it is completed from the defaults of
// Config, a record first reached at that point.
type Outer record {|
    Inner inner = {};
|};

type Inner record {|
    int port = CONFIG.port;
|};

const Config CONFIG = {host: "h"};

type Config record {|
    string host;
    int port = 8080;
|};

public function main() {
    Outer o = {};
    io:println(o); // @output {"inner":{"port":8080}}
    io:println(CONFIG.port); // @output 8080
    io:println(CONFIG == {host: "h", port: 8080}); // @output true
}
