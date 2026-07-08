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

# Used for creating TCP server endpoints. A TCP server endpoint is capable of responding to
# remote callers. The `Listener` is responsible for initializing the endpoint using the
# provided configurations.
public isolated class Listener {

    # Initializes the TCP listener based on the provided configurations.
    # ```ballerina
    # listener tcp:Listener server = check new (8080);
    # ```
    #
    # + localPort - The port number to listen on
    # + config - Configurations related to the `tcp:Listener`. Note: unlike jBallerina's
    #            `*ListenerConfiguration` (an included-record parameter, allowing named-arg-style
    #            `localHost = "x"` at the call site), this is a plain default-valued record
    #            parameter — pass a record literal (`{localHost: "x"}`) instead. See the
    #            README's Notable Behavioural Changes for why.
    public isolated function init(int localPort, ListenerConfiguration config = {}) returns Error? {
        return self.initListener(localPort, config);
    }

    private isolated function initListener(int localPort, ListenerConfiguration config) returns Error? = external;

    # Binds a service to the `tcp:Listener`. Only one service may be attached at a time.
    #
    # + tcpService - The service to attach
    # + name - Ignored; `tcp:Listener` has no path-based routing
    # + return - `()` or else a `tcp:Error` if a service is already attached
    public isolated function attach(Service tcpService, string[]|string? name = ()) returns error? = external;

    # Starts the registered service.
    #
    # + return - An `error` if the listener fails to bind
    public isolated function 'start() returns error? = external;

    # Stops the service listener gracefully. Already-accepted connections are drained before
    # the listener socket closes.
    #
    # + return - An `error` if the listener fails to stop
    public isolated function gracefulStop() returns error? = external;

    # Stops the service listener immediately, force-closing every active connection.
    #
    # + return - An `error` if the listener fails to stop
    public isolated function immediateStop() returns error? = external;

    # Detaches the given service from the `tcp:Listener`.
    #
    # + tcpService - The service to detach
    # + return - `()` or else a `tcp:Error` if the given service isn't the one currently attached
    public isolated function detach(Service tcpService) returns error? = external;
}

// Configurations for the TCP listener.
//
// Fields:
//   localHost    - The hostname or IP address to bind to; defaults to all interfaces.
//   secureSocket - The TLS configurations for the listener.
public type ListenerConfiguration record {|
    string localHost?;
    ListenerSecureSocket secureSocket?;
|};
