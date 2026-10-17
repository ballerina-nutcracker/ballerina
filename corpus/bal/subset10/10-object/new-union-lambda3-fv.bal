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
// jBallerina does not infer anonymous function parameter types against a union of object types either.

class IntFn {
    function init(function (int) returns int callback, int marker) {
        var _ = callback;
        var _ = marker;
    }
}

class FloatFn {
    function init(function (float) returns float callback, string marker) {
        var _ = callback;
        var _ = marker;
    }
}

class Outer {
    function init(IntFn|FloatFn inner, int marker) {
        var _ = inner;
        var _ = marker;
    }
}

class OtherOuter {
    function init(IntFn|FloatFn inner, string marker) {
        var _ = inner;
        var _ = marker;
    }
}

public function main() {
    IntFn|FloatFn f = new (v => v + 1, 1); // @error
    Outer|OtherOuter o = new (new (v => v + 1, 1), 1); // @error
    var _ = f;
    var _ = o;
}
