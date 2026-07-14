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

type Label readonly & distinct object {
    public string value;
};

type NamedLabel readonly & object {
    *Label;

    public string value;
    public string name;
};

type Caller readonly & distinct client object {
    remote function get id() returns int;
};

type CallerImpl readonly & client object {
    *Caller;
};

type PingService readonly & distinct service object {
    resource function get ping() returns string;
};

type PingServiceImpl readonly & service object {
    *PingService;
};

public function main() {
    Label label = object {
        public string value = "x";
    };
    io:println(label is NamedLabel);

    NamedLabel namedLabel = object NamedLabel {
        public string value = "y";
        public string name = "label";
    };
    io:println(namedLabel is Label);

    CallerImpl caller = client object CallerImpl {
        remote function get id() returns int {
            return 7;
        }
    };
    io:println(caller is Caller);

    PingServiceImpl serviceObj = service object PingServiceImpl {
        resource function get ping() returns string {
            return "pong";
        }
    };
    io:println(serviceObj is PingService);
}
