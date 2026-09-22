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

// A worker can appear in any block-bodied anonymous function, including ones
// used as a module variable initializer, a parameter default or an object
// field default.
function () returns int moduleLevel = function() returns int {
    worker w returns int {
        return 6;
    }
    return wait w;
};

function withDefault(function () returns int f = function() returns int {
        worker w returns int {
            return 4;
        }
        return wait w;
    }) returns int {
    return f();
}

class Holder {
    function () returns int make = function() returns int {
        worker w returns int {
            return 9;
        }
        return wait w;
    };

    function call() returns int {
        var f = self.make;
        return f();
    }
}

public function main() {
    io:println(moduleLevel()); // @output 6
    io:println(withDefault()); // @output 4
    Holder holder = new;
    io:println(holder.call()); // @output 9
}
