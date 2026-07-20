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

public function main() returns error? {
    // validate / getVersion / nilAsString / nilAsRecord
    string nilStr = uuid:nilAsString();
    io:println(nilStr); // @output 00000000-0000-0000-0000-000000000000

    boolean nilValid = uuid:validate(nilStr);
    io:println(nilValid); // @output true

    boolean notValid = uuid:validate("not-a-uuid");
    io:println(notValid); // @output false

    uuid:Uuid nilRec = uuid:nilAsRecord();
    io:println(nilRec.timeLow); // @output 0
    io:println(nilRec.node);    // @output 0

    // createType4AsString / validate / getVersion
    string v4 = uuid:createType4AsString();
    io:println(uuid:validate(v4)); // @output true

    uuid:Version ver = check uuid:getVersion(v4);
    io:println(ver == uuid:V4); // @output true

    // createType4AsRecord
    uuid:Uuid v4rec = check uuid:createType4AsRecord();
    io:println(v4rec.timeLow >= 0); // @output true

    // createRandomUuid alias
    string rand = uuid:createRandomUuid();
    io:println(uuid:validate(rand)); // @output true
}
