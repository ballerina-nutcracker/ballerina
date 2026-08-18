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

class First {
    function init(int value = 1) { var _ = value; }
}

class Second {
    function init(int value = 2) { var _ = value; }
}

class NoInit {}

public function main() {
    First|Second ambiguous = new (); // @error ambiguous object type
    First|Second unsuitable = new (name = "x"); // @error failed to find a suitable object type
    NoInit extra = new (1); // @error too many arguments
    object {} anonymous = new (); // @error object type cannot be instantiated without a class
    var _ = ambiguous;
    var _ = unsuitable;
    var _ = extra;
    var _ = anonymous;
}
