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
import ballerina/uuid;

public function main() {
    // createType3AsString (MD5 namespace UUID) — deterministic for same inputs
    string|uuid:Error v3a = uuid:createType3AsString(uuid:NAME_SPACE_DNS, "ballerina.io");
    string|uuid:Error v3b = uuid:createType3AsString(uuid:NAME_SPACE_DNS, "ballerina.io");
    io:println(v3a is string); // @output true
    io:println(v3b is string); // @output true
    if v3a is string && v3b is string {
        io:println(v3a == v3b); // @output true
        io:println(uuid:validate(v3a)); // @output true
        uuid:Version|uuid:Error ver = uuid:getVersion(v3a);
        if ver is uuid:Version {
            io:println(ver == uuid:V3); // @output true
        }
    }

    // createType5AsString (SHA-1 namespace UUID) — deterministic for same inputs
    string|uuid:Error v5a = uuid:createType5AsString(uuid:NAME_SPACE_DNS, "ballerina.io");
    string|uuid:Error v5b = uuid:createType5AsString(uuid:NAME_SPACE_DNS, "ballerina.io");
    io:println(v5a is string); // @output true
    if v5a is string && v5b is string {
        io:println(v5a == v5b); // @output true
        io:println(uuid:validate(v5a)); // @output true
        uuid:Version|uuid:Error ver2 = uuid:getVersion(v5a);
        if ver2 is uuid:Version {
            io:println(ver2 == uuid:V5); // @output true
        }
    }

    // empty name error
    string|uuid:Error emptyErr = uuid:createType3AsString(uuid:NAME_SPACE_DNS, "");
    io:println(emptyErr is uuid:Error); // @output true

    // createType1AsString
    string v1 = uuid:createType1AsString();
    io:println(uuid:validate(v1)); // @output true
    uuid:Version|uuid:Error ver1 = uuid:getVersion(v1);
    if ver1 is uuid:Version {
        io:println(ver1 == uuid:V1); // @output true
    }
}
