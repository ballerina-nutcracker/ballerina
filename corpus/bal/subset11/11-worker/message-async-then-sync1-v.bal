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

// Messages from one worker to another arrive in the order they were sent,
// whether async or sync, even when the receiver takes them late.
public function main() {
    worker a {
        0 -> c;
        1 -> b;
        2 ->> b;
    }
    worker b returns int[] {
        int x = <- c;
        int y = <- a;
        int z = <- a;
        return [x, y, z];
    }
    worker c {
        int k = <- a;
        k + 10 -> b;
    }
    int[] r = wait b;
    io:println(r); // @output [10,1,2]
}
