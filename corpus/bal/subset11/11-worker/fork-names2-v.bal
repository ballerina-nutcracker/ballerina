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

// Forks in an anonymous function are numbered apart from the forks of the
// enclosing function, so a fork after the anonymous function still runs its
// own worker rather than one of an earlier fork with the same worker name.
function forksAroundLambda() returns int {
    int sum = 0;
    if true {
        fork {
            worker w returns int {
                return 1;
            }
        }
        sum += wait w;
    }
    if true {
        fork {
            worker w returns int {
                return 10;
            }
        }
        sum += wait w;
    }
    var g = function() returns int {
        fork {
            worker w returns int {
                return 100;
            }
        }
        return wait w;
    };
    if true {
        fork {
            worker w returns int {
                return 1000;
            }
        }
        sum += wait w;
    }
    return sum + g();
}

public function main() {
    io:println(forksAroundLambda()); // @output 1111
}
