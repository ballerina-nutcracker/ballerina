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

// Fork workers share the variables of the enclosing block, so their
// assignments are visible to the forking worker once it has waited on them.
public function main() {
    int x = 0;
    string s = "a";
    fork {
        worker w {
            x = 5;
        }
        worker v {
            s = "b";
        }
    }
    _ = wait {w, v};
    io:println(x); // @output 5
    io:println(s); // @output b
}
