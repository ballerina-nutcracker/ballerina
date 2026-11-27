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

isolated Listener? defaultListener = ();

# Default HTTP listener port used by the HTTP Default Listener.
# The default value is 9090.
public final int defaultListenerPort = 9090;

# Default HTTP listener configuration used by the HTTP Default Listener.
// The field defaults are written out, since a readonly constructor ignores them (#1103).
public final ListenerConfiguration & readonly defaultListenerConfig = {
    host: "0.0.0.0",
    http1Settings: {keepAlive: KEEPALIVE_AUTO, maxPipelinedRequests: MAX_PIPELINED_REQUESTS},
    secureSocket: (),
    httpVersion: HTTP_2_0,
    timeout: DEFAULT_LISTENER_TIMEOUT,
    gracefulStopTimeout: DEFAULT_GRACEFULSTOP_TIMEOUT,
    socketConfig: {
        connectTimeOut: 15,
        receiveBufferSize: 1048576,
        sendBufferSize: 1048576,
        tcpNoDelay: true,
        socketReuse: true,
        keepAlive: false,
        soBackLog: 100
    },
    http2InitialWindowSize: 65535,
    http2MaxActiveStreams: 100,
    minIdleTimeInStaleState: 300,
    timeBetweenStaleEviction: 30
};

# Returns the default HTTP listener. If the default listener is not already created, a new
# listener will be created with the port and configuration. An error will be returned if
# the listener creation fails.
#
# + return - The default HTTP listener or an error if the listener creation fails.
public isolated function getDefaultListener() returns Listener|ListenerError {
    lock {
        Listener? current = defaultListener;
        if current is Listener {
            return current;
        }
        Listener created = check new (defaultListenerPort, defaultListenerConfig);
        defaultListener = created;
        return created;
    }
}
