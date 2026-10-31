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

// Helpers the native layer calls to build values whose types are declared in this module.

isolated function createError(string kind, string message, error? cause, string? filePath,
        (readonly & byte[])? content)
        returns Error {
    match kind {
        "ConnectionError" => {
            return error ConnectionError(message, cause);
        }
        "FileNotFoundError" => {
            return error FileNotFoundError(message, cause);
        }
        "FileAlreadyExistsError" => {
            return error FileAlreadyExistsError(message, cause);
        }
        "InvalidConfigError" => {
            return error InvalidConfigError(message, cause);
        }
        "ServiceUnavailableError" => {
            return error ServiceUnavailableError(message, cause);
        }
        "AllRetryAttemptsFailedError" => {
            return error AllRetryAttemptsFailedError(message, cause);
        }
        "CircuitBreakerOpenError" => {
            return error CircuitBreakerOpenError(message, cause);
        }
        "ContentBindingError" => {
            return createContentBindingError(message, cause, filePath, content);
        }
    }
    return error Error(message, cause);
}

isolated function createContentBindingError(string message, error? cause, string? filePath,
        (readonly & byte[])? content)
        returns ContentBindingError {
    if filePath is string && content is readonly & byte[] {
        return error ContentBindingError(message, cause, filePath = filePath, content = content);
    }
    if filePath is string {
        return error ContentBindingError(message, cause, filePath = filePath);
    }
    if content is readonly & byte[] {
        return error ContentBindingError(message, cause, content = content);
    }
    return error ContentBindingError(message, cause);
}

isolated function createCaller(ListenerConfiguration config) returns Caller|Error {
    ClientConfiguration clientConfig = {
        protocol: config.protocol,
        host: config.host,
        port: config.port,
        userDirIsRoot: config.userDirIsRoot,
        laxDataBinding: config.laxDataBinding,
        connectTimeout: config.connectTimeout,
        fileTransferMode: config.fileTransferMode,
        sftpCompression: config.sftpCompression
    };
    AuthConfiguration? auth = config.auth;
    if auth is AuthConfiguration {
        clientConfig.auth = auth;
    }
    SocketConfig? socketConfig = config.socketConfig;
    if socketConfig is SocketConfig {
        clientConfig.socketConfig = socketConfig;
    }
    ProxyConfiguration? proxy = config.proxy;
    if proxy is ProxyConfiguration {
        clientConfig.proxy = proxy;
    }
    string? knownHosts = config.sftpSshKnownHosts;
    if knownHosts is string {
        clientConfig.sftpSshKnownHosts = knownHosts;
    }
    FailSafeOptions? csvFailSafe = config.csvFailSafe;
    if csvFailSafe is FailSafeOptions {
        clientConfig.csvFailSafe = csvFailSafe;
    }
    RetryConfig? retryConfig = config.retryConfig;
    if retryConfig is RetryConfig {
        clientConfig.retryConfig = retryConfig;
    }
    Client ftpClient = check new (clientConfig);
    return new (ftpClient);
}

isolated function closeCaller(Caller caller) returns Error? {
    return caller.closeClient();
}

// Stands in for `value:toJsonString`, which is not yet available (ballerina-nutcracker/ballerina#1368).
isolated function toJsonString(json|record {} value) returns string = external;

isolated function fileInfoType() returns typedesc<FileInfo> => FileInfo;

isolated function watchEventType() returns typedesc<WatchEvent> => WatchEvent;
