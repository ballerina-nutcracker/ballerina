// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

// Represents the TCP listener service type.
// A tcp:Service may declare the following remote method:
//   remote function onConnect(Caller caller) returns ConnectionService|Error?;
public type Service distinct service object {
};

// Represents the TCP listener connection service type, returned from onConnect.
// A tcp:ConnectionService may declare the following optional remote methods:
//   remote function onBytes(Caller caller, readonly & byte[] data) returns byte[]|Error?;
//   remote function onError(readonly & Error err) returns Error?;
//   remote function onClose() returns Error?;
public type ConnectionService distinct service object {
};
