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

isolated function spin() returns int {
    int i = 0;
    int acc = 0;
    while i < 5000 {
        acc += i;
        i += 1;
    }
    return acc;
}

// A field initializer belongs to the object's initialization, so it runs
// before `init` starts its named workers. The worker below reads the field
// right away while the declaring strand is still busy in `spin`, so it would
// observe the field unset if the initializer ran after worker startup.
isolated class Holder {
    final int value = spin();
    private int seen = -1;

    isolated function init() {
        worker reader returns int {
            return self.value;
        }
        int observed = wait reader;
        lock {
            self.seen = observed;
        }
    }

    isolated function sawInitializedField() returns boolean {
        lock {
            return self.seen == spin();
        }
    }
}

public function main() {
    Holder h = new;
    io:println(h.sawInitializedField()); // @output true
}
