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
// jBallerina's defaults are written out: a readonly constructor ignores field defaults (#1103), and
// ListenerConfiguration declares host and timeout without theirs.
public final ListenerConfiguration & readonly defaultListenerConfig = {host: "0.0.0.0", timeout: 60, httpVersion: HTTP_2_0};

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
