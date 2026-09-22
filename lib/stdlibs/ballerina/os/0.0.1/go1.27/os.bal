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

// Represents OS module related errors.
public type Error error;

// Process execution error returned when `os:exec` fails.
public type ProcessExecError error;

public class Process {

    # Waits for the subprocess to exit and returns its exit code.
    # Returns `0` for a successful exit; a non-zero value otherwise.
    #
    # + return - Exit code of the subprocess, or an `os:Error` if waiting fails
    public isolated function waitForExit() returns int|Error = external;

    # Returns the output of the subprocess as a byte array.
    # If the process has not yet exited, this call waits for it to finish first.
    #
    # + fileOutputStream - The output stream to read: `io:stdout` (default) or `io:stderr`
    # + return - Output bytes, or an `os:Error` if reading fails
    public isolated function output(io:FileOutputStream fileOutputStream = io:stdout) returns byte[]|Error = external;

    # Terminates the subprocess immediately.
    public isolated function exit() = external;
}

// Represents a command to be executed as a subprocess.
//
// + value - The command name or executable path
// + arguments - Arguments to pass to the command
public type Command record {|
    string value;
    string[] arguments = [];
|};

// Represents additional environment variable overrides for a subprocess.
// Any key-value pair (except `command`) is treated as an environment variable.
public type EnvProperties record {|
    never command?;
    anydata...;
|};

# Returns the value of the environment variable associated with the given name.
# Returns an empty string if the variable is not set.
#
# + name - Name of the environment variable
# + return - Environment variable value, or an empty string if not set
public isolated function getEnv(string name) returns string = external;

# Returns the current user's name.
#
# + return - Current user's name, or an empty string if it cannot be determined
public isolated function getUsername() returns string = external;

# Returns the current user's home directory path.
#
# + return - Current user's home directory, or an empty string if it cannot be determined
public isolated function getUserHome() returns string = external;

# Sets the value of the environment variable identified by `key`.
# The key cannot be an empty string or `"=="`.
#
# + key - Name of the environment variable
# + value - Value to set
# + return - An `os:Error` if the operation fails, otherwise `()`
public isolated function setEnv(string key, string value) returns Error? {
    if key == "" {
        return error Error("The parameter key cannot be an empty string");
    } else if key == "==" {
        return error Error("The parameter key cannot be == sign");
    } else {
        return setEnvExtern(key, value);
    }
}

# Removes the environment variable identified by `key`.
# The key cannot be an empty string.
#
# + key - Name of the environment variable to remove
# + return - An `os:Error` if the operation fails, otherwise `()`
public isolated function unsetEnv(string key) returns Error? {
    if key == "" {
        return error Error("The parameter key cannot be an empty string");
    } else {
        return unsetEnvExtern(key);
    }
}

isolated function setEnvExtern(string key, string value) returns Error? = external;
isolated function unsetEnvExtern(string key) returns Error? = external;

# Returns a map of all environment variables.
#
# + return - Map of environment variable names to their values
public isolated function listEnv() returns map<string> = external;

# Executes a command as a subprocess of the current process.
#
# + command - The command and its arguments
# + envProperties - Additional environment variable overrides for the subprocess
# + return - A `Process` handle on success, or an `os:Error` on failure
public isolated function exec(Command command, *EnvProperties envProperties) returns Process|Error = external;
