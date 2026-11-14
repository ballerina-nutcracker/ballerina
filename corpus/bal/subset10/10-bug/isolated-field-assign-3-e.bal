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

isolated function identity(int[] a) returns int[] => a;

isolated class NonFinalField {
    private int[] arr;

    function init(int[] a) {
        self.arr = a; // @error
    }
}

isolated class DefaultParam {
    private int[] arr;

    isolated function init(int[] a = [1, 2]) {
        self.arr = a; // @error
    }
}

isolated class LocalAlias {
    private final int[] arr;

    isolated function init(int[] a) {
        int[] t = a;
        self.arr = t; // @error
    }
}

isolated class ListConstructor {
    private int[][] arr;

    isolated function init(int[] a) {
        self.arr = [a]; // @error
    }
}

isolated class Conditional {
    private int[] arr;

    isolated function init(boolean b, int[] a) {
        self.arr = b ? a : []; // @error
    }
}

isolated class FunctionCall {
    private int[] arr;

    isolated function init(int[] a) {
        self.arr = identity(a); // @error
    }
}

isolated class CompoundAssignment {
    private xml x = xml ``;

    isolated function init(xml a) {
        self.x += a; // @error
    }
}

isolated class LockLocal {
    private int[] arr = [];

    isolated function set(int a) {
        lock {
            int[] t = [a];
            self.arr = t; // @error
        }
    }
}

isolated service class Service {
    private int[] arr;

    function init(int[] a) {
        self.arr = a; // @error
    }

    resource function get first() returns int {
        lock {
            return self.arr[0];
        }
    }
}

public function main() {
    _ = new NonFinalField([1]);
    _ = new DefaultParam();
    _ = new LocalAlias([1]);
    _ = new ListConstructor([1]);
    _ = new Conditional(true, [1]);
    _ = new FunctionCall([1]);
    _ = new CompoundAssignment(xml `<a/>`);
    _ = new Service([1]);
    LockLocal l = new;
    l.set(1);
}
