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

public function main() {
    from var left in [1, 2]
        join var right in [left] // @error
        on left equals right
        do {};

    from var left in [1, 2]
        join var right in [1, 2]
        on left + right equals right // @error
        do {};

    from var left in [1, 2]
        join var right in [1, 2]
        on left equals right + left // @error
        do {};

    from var value in [1, 2]
        limit value // @error
        do {
            int _ = value;
        };
}
