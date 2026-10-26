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


isolated function initEndpoint(Client clientEndpoint, map<anydata> config) returns Error? = external;

isolated function getBytes(Client clientEndpoint, string path) returns byte[]|Error = external;

isolated function getText(Client clientEndpoint, string path) returns string|Error = external;

isolated function delete(Client clientEndpoint, string path) returns Error? = external;

isolated function append(Client clientEndpoint, InputContent inputContent) returns Error? = external;

isolated function put(Client clientEndpoint, InputContent inputContent) returns Error? = external;

isolated function putJson(Client clientEndpoint, string path, string content, FileWriteOption option) returns Error? = external;

isolated function putXml(Client clientEndpoint, string path, xml content, FileWriteOption option) returns Error? = external;

isolated function putCsv(Client clientEndpoint, string path, string[][]|record {}[] content, FileWriteOption option) returns Error? = external;

isolated function mkdir(Client clientEndpoint, string path) returns Error? = external;

isolated function rmdir(Client clientEndpoint, string path) returns Error? = external;

isolated function rename(Client clientEndpoint, string origin, string destination) returns Error? = external;

isolated function move(Client clientEndpoint, string sourcePath, string destinationPath) returns Error? = external;

isolated function copy(Client clientEndpoint, string sourcePath, string destinationPath) returns Error? = external;

isolated function exists(Client clientEndpoint, string path) returns boolean|Error = external;

isolated function size(Client clientEndpoint, string path) returns int|Error = external;

isolated function list(Client clientEndpoint, string path) returns FileInfo[]|Error = external;

isolated function isDirectory(Client clientEndpoint, string path) returns boolean|Error = external;

isolated function poll(Listener listenerEndpoint) returns Error? = external;

isolated function initListener(Listener listenerEndpoint, ListenerConfiguration config)
        returns Error? = external;

isolated function register(Listener listenerEndpoint, Service ftpService) returns Error? = external;

isolated function deregister(Listener listenerEndpoint, Service ftpService) returns Error? = external;

isolated function cleanup(Listener listenerEndpoint) returns Error? = external;
