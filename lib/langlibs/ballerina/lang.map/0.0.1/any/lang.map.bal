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

# Returns the number of members of a map.
#
# + m - the map
# + return - number of members in `m`
public isolated function length(map<any|error> m) returns int = external;

# Returns a list of all the keys of a map.
#
# + m - the map
# + return - a new list of all keys
public isolated function keys(map<any|error> m) returns string[] = external;
