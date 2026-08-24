// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

isolated int[] values = [42];

client class Client {
    isolated remote function value(int[] xs) returns map<int> {
        return {answer: xs[0]};
    }

    isolated remote function constant() returns map<int> {
        return {answer: 42};
    }
}

function restrictedArgumentFromLock() returns map<int>|error {
    Client c = new;
    lock {
        return trap c->value(values); // @error
    }
}

function restrictedReceiverFromLock() returns map<int>|error {
    Client c = new;
    lock {
        values[0] = 42;
        return trap c->constant(); // @error
    }
}

function unrestrictedValuesFromLock() returns map<int>|error {
    Client c = new;
    int[] localValues = [42];
    lock { // @error
        return trap c->value(localValues);
    }
}

public function main() {
}
