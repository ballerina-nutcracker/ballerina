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

function siblingRefInLoop() {
    fork {
        worker a {
            foreach int _ in 0 ..< 2 {
                int r = wait b; // @error b may be evaluated once per iteration
                io:println(r);
            }
        }
        worker b returns int {
            return 1;
        }
    }
}

function siblingRefInLambda() {
    fork {
        worker a returns int {
            var f = function () returns int {
                return wait d; // @error the lambda may be called more than once
            };
            return f();
        }
        worker d returns int {
            return 1;
        }
    }
}
