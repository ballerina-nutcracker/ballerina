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

// A fork statement is a statement of the forking worker, so it is unreachable
// after that worker panics; it is reported once, not once per worker.
function afterPanic() {
    panic error("stop");
    fork { // @error unreachable fork statement
        worker a {
            1 -> b;
        }
        worker b {
            int _ = <- a;
        }
    }
}

// Likewise after a break in a loop body.
function afterBreak() {
    while true {
        break;
        fork { // @error unreachable fork statement
            worker w {
            }
        }
    }
}
