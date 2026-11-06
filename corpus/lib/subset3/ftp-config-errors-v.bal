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

import ballerina/ftp;
import ballerina/io;

function report(ftp:Client|ftp:Listener|ftp:Error|error result) {
    if result is ftp:InvalidConfigError {
        io:println("InvalidConfigError: ", result.message());
    } else if result is ftp:Error {
        io:println("Error: ", result.message());
    } else if result is error {
        io:println("error: ", result.message());
    } else {
        io:println("created");
    }
}

@ftp:ServiceConfig {path: ""}
service class EmptyPathService {
    *ftp:Service;

    remote function onFileText(string content) returns error? {
        _ = content;
    }
}

@ftp:ServiceConfig {path: "/in", fileNamePattern: "(unclosed"}
service class BadPatternService {
    *ftp:Service;

    remote function onFileText(string content) returns error? {
        _ = content;
    }
}

@ftp:ServiceConfig {path: "/in", fileAgeFilter: {minAge: 10, maxAge: 5}}
service class BadAgeService {
    *ftp:Service;

    remote function onFileText(string content) returns error? {
        _ = content;
    }
}

@ftp:ServiceConfig {path: "/in", fileDependencyConditions: [{targetPattern: "[", requiredFiles: []}]}
service class BadDependencyService {
    *ftp:Service;

    remote function onFileText(string content) returns error? {
        _ = content;
    }
}

service class TooManyParamsService {
    *ftp:Service;

    remote function onFileText(string content, ftp:FileInfo info, ftp:Caller caller, string extra) returns error? {
        _ = content;
        _ = info;
        _ = caller;
        _ = extra;
    }
}

service class RecordXmlService {
    *ftp:Service;

    remote function onFileXml(record {} content) returns error? {
        _ = content;
    }
}

public function main() {
    ftp:RetryConfig[] retries = [
        {count: 0},
        {interval: 0},
        {backOffFactor: 0.5},
        {maxWaitInterval: 0},
        {interval: 5, maxWaitInterval: 2}
    ];
    foreach ftp:RetryConfig retryConfig in retries {
        report(new ftp:Client({retryConfig}));
    }
    // @output InvalidConfigError: retryConfig.count must be greater than 0.
    // @output InvalidConfigError: retryConfig.interval must be greater than 0.
    // @output InvalidConfigError: retryConfig.backOffFactor must be greater than or equal to 1.0.
    // @output InvalidConfigError: retryConfig.maxWaitInterval must be greater than 0.
    // @output InvalidConfigError: retryConfig.maxWaitInterval must be greater than or equal to retryConfig.interval.

    report(new ftp:Client({port: 0}));
    // @output InvalidConfigError: Port must be between 1 and 65535 (got: 0)
    report(new ftp:Client({protocol: ftp:SFTP, port: 65536}));
    // @output InvalidConfigError: Port must be between 1 and 65535 (got: 65536)
    report(new ftp:Listener({port: -21}));
    // @output InvalidConfigError: Port must be between 1 and 65535 (got: -21)

    ftp:CircuitBreakerConfig[] breakers = [
        {failureCategories: []},
        {failureThreshold: 1.5},
        {rollingWindow: {requestVolumeThreshold: 0}},
        {resetTime: 0},
        {rollingWindow: {bucketSize: 0}},
        {rollingWindow: {timeWindow: 5, bucketSize: 10}},
        {rollingWindow: {timeWindow: 25, bucketSize: 10}}
    ];
    foreach ftp:CircuitBreakerConfig circuitBreaker in breakers {
        report(new ftp:Client({port: 1, circuitBreaker}));
    }
    // @output InvalidConfigError: Circuit breaker failureCategories must not be empty
    // @output InvalidConfigError: Circuit breaker failureThreshold must be between 0.0 and 1.0
    // @output InvalidConfigError: Circuit breaker requestVolumeThreshold must be greater than 0
    // @output InvalidConfigError: Circuit breaker resetTime must be greater than 0
    // @output InvalidConfigError: Circuit breaker bucketSize must be greater than 0
    // @output InvalidConfigError: Circuit breaker timeWindow must be greater than or equal to bucketSize
    // @output InvalidConfigError: Circuit breaker timeWindow must be evenly divisible by bucketSize

    report(new ftp:Client({auth: {privateKey: {path: "/key"}}}));
    // @output Error: privateKey can only be used with SFTP protocol.
    report(new ftp:Client({protocol: ftp:SFTP, auth: {secureSocket: {}}}));
    // @output Error: secureSocket can only be used with FTPS protocol.
    report(new ftp:Client({protocol: ftp:SFTP, auth: {privateKey: {path: " "}}}));
    // @output InvalidConfigError: privateKey path cannot be empty.
    report(new ftp:Client({protocol: ftp:FTPS, auth: {secureSocket: {key: {path: "", password: ""}}}}));
    // @output InvalidConfigError: secureSocket.key.path cannot be empty.
    report(new ftp:Client({protocol: ftp:FTPS, auth: {secureSocket: {cert: ""}}}));
    // @output InvalidConfigError: secureSocket.cert.path cannot be empty.
    report(new ftp:Client({protocol: ftp:FTPS, auth: {secureSocket: {cert: "/no/such/cert.pem"}}}));
    // @output Error: Failed to configure SSL/TLS certificates for FTPS: TrustStore PEM file does not exist: /no/such/cert.pem
    report(new ftp:Client({connectTimeout: -1}));
    // @output InvalidConfigError: connectTimeout must be positive or zero (got: -1.0)
    report(new ftp:Client({connectTimeout: 601}));
    // @output InvalidConfigError: connectTimeout must not exceed 600 seconds (got: 601.0)
    report(new ftp:Client({socketConfig: {ftpDataTimeout: -2}}));
    // @output InvalidConfigError: ftpDataTimeout must be positive or zero (got: -2.0)
    report(new ftp:Client({socketConfig: {ftpSocketTimeout: 700}}));
    // @output InvalidConfigError: ftpSocketTimeout must not exceed 600 seconds (got: 700.0)
    report(new ftp:Client({socketConfig: {sftpSessionTimeout: -1}}));
    // @output InvalidConfigError: sftpSessionTimeout must be positive or zero (got: -1.0)
    report(new ftp:Client({proxy: {host: "", port: 8080}}));
    // @output InvalidConfigError: Proxy host cannot be empty
    report(new ftp:Client({proxy: {host: "proxy", port: 0}}));
    // @output InvalidConfigError: Proxy port must be between 1 and 65535 (got: 0)
    report(new ftp:Client({proxy: {host: "proxy", port: 8080}}));
    // @output Error: Proxy configuration is not yet supported
    report(new ftp:Client({protocol: ftp:SFTP, auth: {preferredMethods: []}}));
    // @output Error: No preferred auth method is specified in the SFTP client configurations

    report(new ftp:Listener({auth: {credentials: {username: " "}}}));
    // @output Error: Username cannot be empty
    report(new ftp:Listener({fileNamePattern: "("}));
    // @output InvalidConfigError: Invalid regex pattern for fileNamePattern: error parsing regexp: missing closing ): `(`
    report(new ftp:Listener({protocol: ftp:SFTP, auth: {privateKey: {path: ""}}}));
    // @output Error: Private key path cannot be empty
    report(new ftp:Listener({protocol: ftp:FTPS, auth: {secureSocket: {key: {path: "", password: ""}}}}));
    // @output Error: Failed to load FTPS Server Keystore: Path cannot be empty
    report(new ftp:Listener({protocol: ftp:FTPS, auth: {secureSocket: {cert: {path: "", password: ""}}}}));
    // @output Error: Failed to load FTPS Server Truststore: Path cannot be empty
    report(new ftp:Listener({protocol: ftp:SFTP, auth: {preferredMethods: []}}));
    // @output Error: No preferred auth method is specified in the SFTP client configurations
    report(new ftp:Listener({fileAgeFilter: {minAge: -1}}));
    // @output InvalidConfigError: fileAgeFilter.minAge must be positive or zero (got: -1.0)
    report(new ftp:Listener({fileAgeFilter: {maxAge: -3}}));
    // @output InvalidConfigError: fileAgeFilter.maxAge must be positive or zero (got: -3.0)
    report(new ftp:Listener({fileAgeFilter: {minAge: 10, maxAge: 1}}));
    // @output InvalidConfigError: fileAgeFilter.minAge (10.0) must not exceed fileAgeFilter.maxAge (1.0)
    report(new ftp:Listener({connectTimeout: 1000}));
    // @output InvalidConfigError: connectTimeout must not exceed 600 seconds (got: 1000.0)
    report(new ftp:Listener({retryConfig: {count: -1}}));
    // @output InvalidConfigError: retryConfig.count must be greater than 0.

    ftp:Listener|ftp:Error ftpListener = new ({port: 1});
    if ftpListener is ftp:Listener {
        error? polled = ftpListener.poll();
        io:println(polled is error ? polled.message() : "polled"); // @output FTP listener is not initialized. Attach a service before polling.
        ftp:Service[] services = [new EmptyPathService(), new BadPatternService(), new BadAgeService(),
            new BadDependencyService(), new TooManyParamsService(), new RecordXmlService()];
        foreach ftp:Service svc in services {
            error? attached = ftpListener.attach(svc);
            io:println(attached is error ? attached.message() : "attached");
        }
        // @output The 'path' field cannot be empty in @ftp:ServiceConfig annotation.
        // @output Invalid regex pattern '(unclosed' in @ftp:ServiceConfig.fileNamePattern: error parsing regexp: missing closing ): `(unclosed`
        // @output fileAgeFilter.minAge (10.0) must not exceed fileAgeFilter.maxAge (5.0)
        // @output Invalid regex pattern '[' in fileDependencyConditions.targetPattern: error parsing regexp: missing closing ]: `[`
        // @output Too many parameters for 'onFileText'. Format-specific handlers accept at most 3 parameters: (content, fileInfo?, caller?).
        // @output Binding XML content to a record type is not yet supported
        error? attachedWithNames = ftpListener.attach(new EmptyPathService(), ["inbox", "orders"]);
        io:println(attachedWithNames is error ? attachedWithNames.message() : "attached");
        // @output The 'path' field cannot be empty in @ftp:ServiceConfig annotation.
    }
}
