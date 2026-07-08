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

type OpenEmpty record {
};

type ClosedEmpty record {|
|};

type RequiredX record {
    int x;
};

type RequiredY record {
    int y;
};

type RequiredZ record {
    int z;
};

type XAndY RequiredX & RequiredY;

function intReceiver() {
    int i = 1;
    _ = i?.x; // @error
}

function listReceiver() {
    int[] l = [1];
    _ = l?.x; // @error
}

function notSubtypeOfMappingOrNil() {
    int|RequiredX value = {x: 1};
    _ = value?.x; // @error
}

function unionWithNoField() {
    RequiredY|RequiredZ value = {y: 1};
    _ = value?.x; // @error
}

function openRecordNoField() {
    OpenEmpty value = {};
    value["x"] = 1;
    _ = value?.x; // @error
}

function closedRecordNoField() {
    ClosedEmpty value = {};
    _ = value?.x; // @error
}

function intersectionWithNoSuchField() {
    XAndY value = {x: 1, y: 2};
    _ = value?.z; // @error
}

