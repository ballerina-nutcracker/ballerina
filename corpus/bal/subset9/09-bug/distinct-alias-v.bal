// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import ballerina/io;

type ParentError distinct error;
type ChildError distinct ParentError;

type ParentObject distinct object {
    function id() returns int;
};

type ChildObject distinct ParentObject;

class ParentObjectImpl {
    *ParentObject;

    function id() returns int {
        return 0;
    }
}

class ChildObjectImpl {
    *ChildObject;

    function id() returns int {
        return 1;
    }
}

public function main() {
    ChildError childError = error ChildError("child");
    io:println(childError is ParentError); // @output true

    ParentError parentError = error ParentError("parent");
    io:println(parentError is ChildError); // @output false

    ChildObject childObject = new ChildObjectImpl();
    io:println(childObject is ParentObject); // @output true

    ParentObject parentObject = new ParentObjectImpl();
    io:println(parentObject is ChildObject); // @output false
}
