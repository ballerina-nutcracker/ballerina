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

public isolated class Listener {
    private final string path;
    private final boolean recursive;

    public isolated function init(string path, boolean recursive = false) returns error? {
        self.path = path;
        self.recursive = recursive;
    }

    public isolated function describe() returns string {
        return string `${self.path}:${self.recursive}`;
    }
}

public function main() returns error? {
    Listener named = check new (path = "/tmp", recursive = true);
    io:println(named.describe()); // @output /tmp:true

    Listener mixed = check new ("/var", recursive = true);
    io:println(mixed.describe()); // @output /var:true

    Listener defaulted = check new (path = "/etc");
    io:println(defaulted.describe()); // @output /etc:false
}
