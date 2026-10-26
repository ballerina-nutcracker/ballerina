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
import ballerina/os;

type Order record {|
    int id;
    string region;
    int amount;
|};

type RegionAmounts record {|
    string region;
    int[] amounts;
|};

function scale() returns int {
    string raw = os:getEnv("BAL_SCALE");
    if raw == "" {
        return 1;
    }
    int|error n = int:fromString(raw);
    if n is int && n > 0 {
        return n;
    }
    panic error("BAL_SCALE must be a positive integer: " + raw);
}

public function main() {
    int n = 100000 * scale();
    string[] regions = ["north", "south", "east", "west"];
    Order[] orders = [];
    foreach int i in 0 ..< n {
        orders.push({id: i, region: regions[i % 4], amount: (i * 7919) % 1000});
    }
    Order[] large = from Order o in orders
        where o.amount >= 500
        order by o.amount descending, o.id
        select o;
    RegionAmounts[] totals = from Order o in orders
        let int amount = o.amount
        group by string region = o.region
        select {region, amounts: [amount]};
    int grand = 0;
    foreach RegionAmounts t in totals {
        foreach int amount in t.amounts {
            grand += amount;
        }
    }
    io:println(large.length(), " ", large[0].amount, " ", totals.length(), " ", grand); // @output 50000 999 4 49950000
}
