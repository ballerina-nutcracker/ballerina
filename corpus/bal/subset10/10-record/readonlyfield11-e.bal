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

type Opt record {|
    readonly int value?;
    int otherValue;
|};

type Base record {|
    readonly int value;
|};

type Derived record {|
    *Base;
    int otherValue;
|};

type Open record {
    readonly int value;
};

type MR record {|
    readonly int value;
    int otherValue;
|};

public function main() {
    Opt o = {value: 1, otherValue: 2};
    o.value = 3; // @error readonly optional field

    Derived d = {value: 1, otherValue: 2};
    d.value = 5; // @error readonly field through type inclusion

    Open op = {value: 1};
    op.value = 2; // @error readonly field in an open record

    MR|int x = 5;
    if x is MR {
        x.value = 1; // @error readonly field through a narrowed type
    }
}
