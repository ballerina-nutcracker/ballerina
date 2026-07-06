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

class TaggedListener {
    string tag;

    function init(string tag) {
        self.tag = tag;
    }

    public function attach(service object {} svc, () attachPoint = ()) returns () {
        var _ = svc;
        var _ = attachPoint;
        io:println("attached: ", self.tag);
    }

    public function detach(service object {} svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }
}

function makeListener(string tag) returns TaggedListener {
    return new TaggedListener(tag);
}

listener TaggedListener l = new ("var");

service on l, new TaggedListener("inline"), makeListener("call") {
    // @output attached: var
    // @output attached: inline
    // @output attached: call
}

public function main() {
}
