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

public class FailingIterator {
    public isolated function next() returns record {|int value;|}|error? {
        return error("iterator completion failed");
    }
}

class FailingIterable {
    *object:Iterable;

    public function iterator() returns FailingIterator {
        return new;
    }
}

public function main() {
    () result = from var value in new FailingIterable()
        do { // @error
            int _ = value;
        };
    _ = result;

    () joinResult = from var left in [1]
        join var right in new FailingIterable()
        on left equals right
        do { // @error
            int _ = left + right;
        };
    _ = joinResult;
}
