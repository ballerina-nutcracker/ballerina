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

# The HTTP request interceptor service object type
public type RequestInterceptor distinct service object {

};

# The HTTP response interceptor service object type
public type ResponseInterceptor distinct service object {

};

# The HTTP request error interceptor service object type
public type RequestErrorInterceptor distinct service object {

};

# The HTTP response error interceptor service object type
public type ResponseErrorInterceptor distinct service object {

};

# The service type to be used when engaging interceptors at the service level
public type InterceptableService distinct service object {
    *Service;

    # Function to define interceptor pipeline
    #
    # + return - The `http:Interceptor|http:Interceptor[]`
    public function createInterceptors() returns Interceptor|Interceptor[];
};

# The return type of an interceptor service function
public type NextService RequestInterceptor|ResponseInterceptor|Service;

# Types of HTTP interceptor services
public type Interceptor RequestInterceptor|ResponseInterceptor|RequestErrorInterceptor|ResponseErrorInterceptor;

// The dispatcher reads this signature to learn the interceptor types, which a native cannot name
// directly.
isolated function interceptorTypes(InterceptableService interceptableService, RequestInterceptor requestInterceptor,
        ResponseInterceptor responseInterceptor, RequestErrorInterceptor requestErrorInterceptor,
        ResponseErrorInterceptor responseErrorInterceptor, NextService nextService, Response response)
        returns [InterceptableService, RequestInterceptor, ResponseInterceptor, RequestErrorInterceptor,
        ResponseErrorInterceptor, NextService, Response] =>
    [interceptableService, requestInterceptor, responseInterceptor, requestErrorInterceptor, responseErrorInterceptor,
            nextService, response];
