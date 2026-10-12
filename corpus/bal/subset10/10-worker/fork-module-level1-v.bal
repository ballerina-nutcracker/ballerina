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

// Anonymous functions in a module-level initializer, a default parameter and an
// object field initializer may contain a fork whose workers exchange no
// messages; only message actions are unsupported there.
final function () returns int moduleInit = function() returns int {
    fork {
        worker w returns int {
            return 1;
        }
    }
    return wait w;
};

final int fromModuleInit = moduleInit();

class Holder {
    function () returns int fieldInit = function() returns int {
        fork {
            worker w returns int {
                return 20;
            }
        }
        return wait w;
    };
}

function withDefault(function () returns int f = function() returns int {
        fork {
            worker w returns int {
                return 300;
            }
        }
        return wait w;
    }) returns int {
    return f();
}

public function main() {
    io:println(fromModuleInit); // @output 1
    Holder h = new;
    function () returns int fieldInit = h.fieldInit;
    io:println(fieldInit()); // @output 20
    io:println(withDefault()); // @output 300
}
