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

# Represents a service listener that monitors the email server location.
public class PopListener {

    # Gets invoked during the `email:PopListener` initialization.
    #
    # + listenerConfig - Configurations for the email endpoint
    # + return - `()` or else an `email:Error` upon failure to
    #            initialize the listener
    public isolated function init(PopListenerConfiguration listenerConfig) returns Error? {
        PopConfiguration popConfig = {
            port: listenerConfig.port,
            security: listenerConfig.security
        };
        SecureSocket? secureSocketParam = listenerConfig?.secureSocket;
        if secureSocketParam !is () {
            popConfig.secureSocket = secureSocketParam;
        }
        PopClient popClient = check new (listenerConfig.host, listenerConfig.username, listenerConfig.password,
            popConfig);
        initListener(self, popClient, listenerConfig.pollingInterval);
    }

    # Starts the `email:PopListener`.
    # ```ballerina
    # email:Error? result = emailListener.start();
    # ```
    #
    # + return - `()` or else an `email:Error` upon failure to
    #            start the listener
    public isolated function 'start() returns error? {
        return toEmailError(startListener(self));
    }

    # Binds a service to the `email:PopListener`.
    # ```ballerina
    # email:Error? result = emailListener.attach(helloService, hello);
    # ```
    #
    # + s - Type descriptor of the service
    # + name - Name of the service
    # + return - `()` or else a `email:Error` upon failure to register the listener
    public isolated function attach(Service s, string[]|string? name = ()) returns error? {
        _ = name;
        register(self, s);
    }

    # Stops consuming messages and detaches the service from the `email:PopListener`.
    # ```ballerina
    # email:Error? result = emailListener.detach(helloService);
    # ```
    #
    # + s - Type descriptor of the service
    # + return - `()` or else a `email:Error` upon failure to detach the service
    public isolated function detach(Service s) returns error? {
        deregister(self, s);
    }

    # Stops the `email:PopListener` forcefully.
    # ```ballerina
    # email:Error? result = emailListener.immediateStop();
    # ```
    #
    # + return - `()` or else a `email:Error` upon failure to stop the listener
    public isolated function immediateStop() returns error? {
        stopListener(self, false);
    }

    # Stops the `email:PopListener` gracefully.
    # ```ballerina
    # email:Error? result = emailListener.gracefulStop();
    # ```
    #
    # + return - () or else error upon failure to stop the listener
    public isolated function gracefulStop() returns error? {
        stopListener(self, true);
    }

    # Registers for the Email service.
    # ```ballerina
    # emailListener.register(helloService, hello);
    # ```
    #
    # + emailService - Type descriptor of the service
    # + name - Service name
    public isolated function register(Service emailService, string? name) {
        _ = name;
        register(self, emailService);
    }
}

# Configuration for Email listener endpoint.
#
# + host - Email server host
# + username - Email server access username
# + password - Email server access password
# + pollingInterval - Periodic time interval (in seconds) to check new update
# + port - Port number of the POP server
# + security - Type of security channel
# + secureSocket - Secure socket configuration
public type PopListenerConfiguration record {|
    string host;
    string username;
    string password;
    decimal pollingInterval = 30;
    int port = 995;
    Security security = SSL;
    SecureSocket secureSocket?;
|};

isolated function initListener(PopListener|ImapListener listenerEndpoint, PopClient|ImapClient mailClient,
        decimal pollingInterval) = external;

isolated function startListener(PopListener|ImapListener listenerEndpoint) returns error? = external;

isolated function stopListener(PopListener|ImapListener listenerEndpoint, boolean graceful) = external;

isolated function register(PopListener|ImapListener listenerEndpoint, Service emailService) = external;

isolated function deregister(PopListener|ImapListener listenerEndpoint, Service emailService) = external;
