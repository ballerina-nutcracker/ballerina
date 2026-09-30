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

import ballerina/crypto;


# Represents the caller object passed to tcp service remote methods.
#
# + remoteHost - The hostname or the IP address of the remote host
# + remotePort - The port number of the remote host
# + localHost - The bound hostname
# + localPort - The port number to which the socket is bound
# + id - The unique ID associated with the connection
public isolated client class Caller {

    public final string remoteHost;
    public final int remotePort;
    public final string localHost;
    public final int localPort;
    public final string id;

    // Package-level private init to prevent object creation from user code.
    // Every Caller instance is constructed by the listener's native code.
    isolated function init(string remoteHost, int remotePort, string localHost, int localPort, string id) {
        self.remoteHost = remoteHost;
        self.remotePort = remotePort;
        self.localHost = localHost;
        self.localPort = localPort;
        self.id = id;
    }

    # Sends the given data to the connected remote host.
    #
    # + data - The data to send to the remote host
    # + return - `()` or else a `tcp:Error` if the given data cannot be sent
    isolated remote function writeBytes(byte[] data) returns Error? = external;

    # Closes the connection.
    #
    # + return - `()` or else a `tcp:Error` if the connection cannot be properly closed
    isolated remote function close() returns Error? = external;
}


// Represents TCP module related errors.
// Note: distinct error types are not yet supported; Error is currently an alias for error.
public type Error error;


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



// Secure socket configuration for the TCP client.
//
// Not supported: crypto:TrustStore (PKCS12) as `cert` — a PEM certificate file path string is
// fully supported instead; supplying a crypto:TrustStore value returns an Error.
//
// Fields:
//   enable           - Enable SSL validation.
//   cert             - A PEM certificate file path that the client trusts (crypto:TrustStore not supported).
//   protocol         - SSL/TLS protocol related options.
//   ciphers          - List of ciphers to be used.
//   handshakeTimeout - SSL handshake timeout, in seconds.
//   sessionTimeout   - SSL session timeout, in seconds.
public type ClientSecureSocket record {|
    boolean enable = true;
    crypto:TrustStore|string cert?;
    record {|
        Protocol name;
        string[] versions = [];
    |} protocol?;
    string[] ciphers?;
    decimal handshakeTimeout?;
    decimal sessionTimeout?;
|};

// Secure socket configuration for the TCP listener.
//
// Not supported: crypto:KeyStore (PKCS12) as `key` — a CertKey (certificate + private key file
// pair) is fully supported instead; supplying a crypto:KeyStore value returns an Error.
//
// Fields:
//   key              - The server certificate and private key (crypto:KeyStore not supported).
//   protocol         - SSL/TLS protocol related options.
//   ciphers          - List of ciphers to be used.
//   handshakeTimeout - SSL handshake timeout, in seconds.
//   sessionTimeout   - SSL session timeout, in seconds.
public type ListenerSecureSocket record {|
    crypto:KeyStore|CertKey key;
    record {|
        Protocol name;
        string[] versions = [];
    |} protocol?;
    string[] ciphers = [];
    decimal handshakeTimeout?;
    decimal sessionTimeout?;
|};

// Represents a combination of a certificate and its private key.
//
// Fields:
//   certFile    - A file containing the certificate.
//   keyFile     - A file containing the private key in PKCS8 format.
//   keyPassword - Password of the private key if it is encrypted.
public type CertKey record {|
    string certFile;
    string keyFile;
    string keyPassword?;
|};

// Represents protocol options.
public enum Protocol {
    SSL,
    TLS
}


// Represents the TCP listener service type.
// A tcp:Service may declare the following remote method:
//   remote function onConnect(Caller caller) returns ConnectionService|Error?;
public type Service service object {
};

// Represents the TCP listener connection service type, returned from onConnect.
// A tcp:ConnectionService may declare the following optional remote methods:
//   remote function onBytes(Caller caller, readonly & byte[] data) returns byte[]|Error?;
//   remote function onError(readonly & Error err) returns Error?;
//   remote function onClose() returns Error?;
public type ConnectionService service object {
};


# Initializes the TCP connection client based on the provided configurations.
public isolated client class Client {

    # Initializes the TCP client based on the provided configurations.
    # ```ballerina
    # tcp:Client socketClient = check new ("www.remote.com", 80, {localHost: "localHost"});
    # ```
    #
    # + remoteHost - The hostname or the IP address of the remote host
    # + remotePort - The port number of the remote host
    # + config - Connection-oriented client-related configurations. Note: unlike jBallerina's
    #            `*ClientConfiguration` (an included-record parameter, allowing named-arg-style
    #            `localHost = "x"` at the call site), this is a plain default-valued record
    #            parameter — pass a record literal (`{localHost: "x"}`) instead. See the
    #            README's Notable Behavioural Changes for why.
    public isolated function init(string remoteHost, int remotePort, ClientConfiguration config = {}) returns Error? {
        return self.initTcpConnection(remoteHost, remotePort, config);
    }

    private isolated function initTcpConnection(string remoteHost, int remotePort, ClientConfiguration config)
        returns Error? = external;

    # Sends the given data to the connected remote host.
    # ```ballerina
    # tcp:Error? result = socketClient->writeBytes("msg".toBytes());
    # ```
    #
    # + data - The data that needs to be sent to the connected remote host
    # + return - `()` or else a `tcp:Error` if the given data cannot be sent
    isolated remote function writeBytes(byte[] data) returns Error? = external;

    # Reads data from the connected remote host.
    # ```ballerina
    # (readonly & byte[])|tcp:Error result = socketClient->readBytes();
    # ```
    #
    # + return - The `readonly & byte[]` or else a `tcp:Error` if the data cannot be read from the remote host
    isolated remote function readBytes() returns (readonly & byte[])|Error = external;

    # Closes the connection.
    # ```ballerina
    # tcp:Error? closeResult = socketClient->close();
    # ```
    #
    # + return - A `tcp:Error` if it cannot close the connection or else `()`
    isolated remote function close() returns Error? = external;
}

// Configurations for the connection-oriented TCP client.
//
// Fields:
//   localHost    - Local binding interface hostname or IP address.
//   timeout      - The socket read timeout, in seconds. Defaults to 300 seconds (5 minutes).
//   writeTimeout - The socket write timeout, in seconds. Defaults to 300 seconds (5 minutes).
//   secureSocket - The TLS configurations for the client.
public type ClientConfiguration record {|
    string localHost?;
    decimal timeout = 300;
    decimal writeTimeout = 300;
    ClientSecureSocket secureSocket?;
|};
