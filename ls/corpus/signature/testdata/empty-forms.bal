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

class Box {
    function init(int size = 1) { _ = size; }
    function put(int count = 1, string title = "x") { _ = count; _ = title; }
}
client class Client {
    remote function send(int count = 1, string title = "x") { _ = count; _ = title; }
}
class Empty {}
public function main() {
    Box box = new Box();
    Box other = new ();
    box.put();
    Client peer = new;
    peer->send();
    Empty empty = new Empty();
    _ = other; _ = empty;
}
