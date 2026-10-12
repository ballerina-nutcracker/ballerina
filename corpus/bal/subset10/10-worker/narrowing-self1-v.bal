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

type Named object {
    function name() returns string;
};

// self is implicitly final, so a worker that captures it cannot assign it and
// the narrowing of self established before the workers survives them.
class Box {
    int n = 1;

    function describe() returns string {
        if self !is Named {
            return "unnamed";
        }
        worker w returns int {
            return self.n;
        }
        int n = wait w;
        return self.name() + n.toString();
    }
}

public function main() {
    Box b = new;
    io:println(b.describe()); // @output unnamed
}
