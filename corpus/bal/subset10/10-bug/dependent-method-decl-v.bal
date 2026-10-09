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

class Store {
    function get(string key, typedesc<anydata> td = <>) returns td|error = external;
}

client class Client {
    resource function get items/[int id](typedesc<anydata> targetType = <>) returns targetType|error = external;

    remote function fetch(typedesc<anydata> targetType = <>) returns targetType|error = external;
}

function useStore(Store s) returns error? {
    int _ = check s.get("key");
}

function useClient(Client c) returns error? {
    string _ = check c->/items/[1]();
    int _ = check c->fetch();
}

public function main() {
    Store _ = new;
    Client _ = new;
    io:println("declared"); // @output declared
}
