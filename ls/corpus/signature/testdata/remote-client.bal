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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

client class MyClient {

    remote function remote1(string id, int age, string name) {

    }

    resource function get users() returns string {
        return "";
    }

    resource function post .(string a, int b) {
    }

    resource function get [string... ids]() {

    }

    resource function get name/[string id1]() {

    }

    resource function get users/[string id1]/name() returns int {
        return 0;
    }

    resource function post users/[string id1]/names/[string... ids](string b, string... ids2) {

    }

    resource function post users/[string id1]/alias(string b, string... ids2) {

    }
}

public function test() {
    MyClient cl = new ();
    cl -> /users/abc/alias.post();
    cl -> /users/path1/names/path1.post("hello", "world", );
    cl-> remote1();
    cl-> remote1("hello",);
}
