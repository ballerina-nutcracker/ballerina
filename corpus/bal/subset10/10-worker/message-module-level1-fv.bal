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

class IntFn {
    int result;

    function init(function () returns int callback) {
        self.result = callback();
    }
}

class StringFn {
    string result;

    function init(function () returns string callback) {
        self.result = callback();
    }
}

// Message actions are supported only in the bodies of module-level functions
// and methods. These bodies are type resolved with module-level nodes.
IntFn|StringFn moduleLevel = new (function () returns int {
    worker w {
        1 -> function; // @error
    }
    return <- w; // @error
});

function withDefault(function () returns int f = function() returns int {
        worker w {
            2 -> function; // @error
        }
        return <- w; // @error
    }) returns int {
    return f();
}

class Holder {
    function () returns int make = function() returns int {
        worker w {
            3 ->> function; // @error
        }
        return <- w; // @error
    };
}

type R record {
    function () returns int make = function() returns int {
        worker w returns error? {
            4 -> function; // @error
            check flush; // @error
        }
        return <- w; // @error
    };
};

public function main() {
    IntFn|StringFn m = moduleLevel;
    if m is IntFn {
        io:println(m.result);
    }
    io:println(withDefault());
    Holder holder = new;
    var make = holder.make;
    io:println(make());
    R r = {};
    io:println(r.make());
}
