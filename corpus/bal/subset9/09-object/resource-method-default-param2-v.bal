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

client class Store {
    resource function get price(int id = 10) returns int|error {
        return id;
    }

    resource function get total(int price, int tax = price / 10) returns int|error {
        return price + tax;
    }
}

public function main() returns error? {
    Store store = new;
    int defaultPrice = check store->/price.get();
    int defaultTax = check store->/total.get(40);
    int explicitTax = check store->/total.get(40, 5);
    io:println(defaultPrice); // @output 10
    io:println(defaultTax); // @output 44
    io:println(explicitTax); // @output 45
}
