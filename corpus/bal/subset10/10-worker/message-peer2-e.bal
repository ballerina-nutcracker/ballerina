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

// A message action in a lambda runs on the lambda's own default worker, which
// isn't a peer of any worker of the enclosing function.
function throughLambda() {
    worker a {
        int _ = <- function;
    }
    var f = function () {
        1 -> a; // @error worker interactions are only allowed between peers
    };
    f();
}
