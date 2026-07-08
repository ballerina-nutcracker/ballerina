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

public type ConfigService isolated service object {
    function getX() returns int;
};

class SimpleListener {
    private ConfigService? s = ();

    public function attach(ConfigService svc, () attachPoint = ()) returns () {
        var _ = svc;
        var _ = attachPoint;
        self.s = svc;
    }

    public function detach(ConfigService svc) returns error? {
        var _ = svc;
        self.s = ();
    }

    public function 'start() returns error? {
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }

    function getX() returns int {
        var svc = self.s;
        if svc == () {
            panic error("no service");
        }
        return svc.getX();
    }
}

listener SimpleListener l = new ();

isolated service on l {
    public final int x = 10;

    function getX() returns int {
        return self.x;
    }
}

public function main() {
    io:println(l.getX()); // @output 10
}
