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

# Represents the default connection timeout values for the oauth2 client.
public const decimal DEFAULT_CONNECT_TIMEOUT = 15;

# Represents the default request timeout value for the oauth2 client.
public const decimal DEFAULT_REQ_TIMEOUT = 30;

# Represents the maximum time(in seconds) to wait for a connection to be established with the oauth2 endpoint.
# Defaults to 15 seconds. This is a global configuration which will be applied to all the internal oauth2 client calls.
// `configurable` module variables are not supported yet (#781), so the global timeouts keep their defaults.
public final decimal globalConnectTimeout = DEFAULT_CONNECT_TIMEOUT;

# Represents the maximum time(in seconds) to wait for a response before the oauth2 endpoint request times out.
# Defaults to 30 seconds. This is a global configuration which will be applied to all the internal oauth2 client calls.
public final decimal globalReqTimeout = DEFAULT_REQ_TIMEOUT;
