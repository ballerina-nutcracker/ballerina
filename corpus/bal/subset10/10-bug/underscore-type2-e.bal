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

type Alias _; // @error
type Array _[]; // @error
type Optional _?; // @error
type Union int|_; // @error
type Tuple [int, _]; // @error
type Mapping map<_>; // @error
type Parenthesized (_); // @error
type Qualified io:_; // @error
type Predeclared int:_; // @error
type QuotedAlias '_; // @error
type QuotedArray '_[]; // @error
type QuotedQualified io:'_; // @error
type QuotedPredeclared int:'_; // @error

_ moduleValue = 1; // @error

function acceptValue(_ value) { // @error
}

public function main() {
    _ localValue = 1; // @error
    _[] arrayValue = []; // @error
    int|_ unionValue = 1; // @error
    [int, _] tupleValue = [1, 2]; // @error
    io:_ qualifiedValue = 1; // @error
    transaction:_ transactionValue = 1; // @error
    '_ quotedValue = 1; // @error
    io:'_ quotedQualifiedValue = 1; // @error
    transaction:'_ quotedTransactionValue = 1; // @error
}
