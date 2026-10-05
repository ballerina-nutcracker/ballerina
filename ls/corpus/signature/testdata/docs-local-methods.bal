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
    # Local constructor.
    # + size - Initial size.
    function init(int size) { _ = size; }
    # Local method.
    # + value - Stored value.
    function put(int value) { _ = value; }
}
client class Client {
    # Local remote method.
    # + value - Sent value.
    remote function send(int value) { _ = value; }
}
public function main() {
    Box box = new Box(1);
    box.put(2);
    Client peer = new;
    peer->send(3);
}
