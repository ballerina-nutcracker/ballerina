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

// Resource declarations are allowed in network interaction objects (§5.6.3.5).
type PathId int;

type ServiceObject service object {
    resource function get name() returns string;
};

type ClientObject client object {
    resource function get name() returns string;
};

type ClientPaths client object {
    resource function get .() returns string;
    resource function get items/[PathId id](int count) returns string;
    resource function get files/[string... tail]() returns string;
    resource function post items/[int](string value) returns int;
    resource function get users() returns string;
    function get() returns int;
};

type IncludedClient client object {
    *ClientObject;
};

public function main() {
}
