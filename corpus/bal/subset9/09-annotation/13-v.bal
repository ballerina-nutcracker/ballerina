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

type Info record {|
    int code;
|};

annotation Info info on type;
annotation Info[] infos on type;

function code() returns int {
    return 1;
}

@info {code: code()}
type Target record {|
    string id;
|};

@infos {code: 2}
@infos {code: code()}
type RepeatedTarget record {|
    string id;
|};

Info? moduleMetadata = Target.@info;

public function main() {
    if moduleMetadata is Info {
        io:println(moduleMetadata.code); // @output 1
    }

    Info? metadata = Target.@info;
    if metadata is Info {
        io:println(metadata.code); // @output 1
    }

    Info[]? repeatedMetadata = RepeatedTarget.@infos;
    if repeatedMetadata is Info[] {
        io:println(repeatedMetadata[0].code); // @output 2
        io:println(repeatedMetadata[1].code); // @output 1
    }
}
