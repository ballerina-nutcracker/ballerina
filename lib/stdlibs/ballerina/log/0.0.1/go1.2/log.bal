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

// Represents log module errors.
// Note: distinct error types are not yet supported; Error is currently an alias for error.
public type Error error;

# Log level types.
public enum Level {
    DEBUG,
    ERROR,
    INFO,
    WARN
}

# Supported log formats.
public enum LogFormat {
    # The JSON log format.
    JSON_FORMAT = "json",
    # The Logfmt log format.
    LOGFMT = "logfmt"
}

// Key-value pairs to be included in the log record.
// The keys `msg` and `'error` are reserved and cannot be used.
public type KeyValues record {|
    never msg?;
    never 'error?;
    anydata...;
|};

# Prints debug logs.
#
# + msg - The message to be logged
# + 'error - The error to be logged
# + keyValues - Additional key-value pairs to be logged
public isolated function printDebug(string msg, error? 'error = (), *KeyValues keyValues) {
    externPrintLog("DEBUG", msg, 'error, keyValues);
}

# Prints error logs.
#
# + msg - The message to be logged
# + 'error - The error to be logged
# + keyValues - Additional key-value pairs to be logged
public isolated function printError(string msg, error? 'error = (), *KeyValues keyValues) {
    externPrintLog("ERROR", msg, 'error, keyValues);
}

# Prints info logs.
#
# + msg - The message to be logged
# + 'error - The error to be logged
# + keyValues - Additional key-value pairs to be logged
public isolated function printInfo(string msg, error? 'error = (), *KeyValues keyValues) {
    externPrintLog("INFO", msg, 'error, keyValues);
}

# Prints warn logs.
#
# + msg - The message to be logged
# + 'error - The error to be logged
# + keyValues - Additional key-value pairs to be logged
public isolated function printWarn(string msg, error? 'error = (), *KeyValues keyValues) {
    externPrintLog("WARN", msg, 'error, keyValues);
}

isolated function externPrintLog(string level, string msg, error? 'error, KeyValues keyValues) = external;
