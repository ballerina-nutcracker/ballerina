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

# Contains the configurations for an HTTP service.
# The `auth` and `serviceType` fields of jBallerina's `HttpServiceConfig` are not supported in this implementation.
#
# + host - Domain name of the service
# + compression - The status of compression
# + chunking - Configures the chunking behaviour for the service
# + cors - The cross origin resource sharing configurations for the service
# + mediaTypeSubtypePrefix - Service specific media-type subtype prefix
# + treatNilableAsOptional - Treat Nilable parameters as optional
# + openApiDefinition - The generated OpenAPI definition for the HTTP service. This is auto-generated at compile-time if OpenAPI doc auto generation is enabled
# + validation - Enables the inbound payload validation functionality which provided by the constraint package. Enabled by default
# + basePath - Base path to be used with the service implementation. This is only allowed on service contract types
# + laxDataBinding - Enables or disables relaxed data binding on the service side. Disabled by default.
#                    When enabled, the JSON data will be projected to the Ballerina record type and during the projection,
#                    nil values will be considered as optional fields and absent fields will be considered for nilable types
public type HttpServiceConfig record {|
    string host = "b7a.default";
    CompressionConfig compression = {};
    Chunking chunking = CHUNKING_AUTO;
    CorsConfig cors = {};
    string mediaTypeSubtypePrefix?;
    boolean treatNilableAsOptional = true;
    byte[] openApiDefinition = [];
    boolean validation = true;
    string basePath?;
    boolean laxDataBinding = false;
|};

# Configurations for CORS support.
#
# + allowHeaders - The array of allowed headers by the service
# + allowMethods - The array of allowed methods by the service
# + allowOrigins - The array of origins with which the response is shared by the service
# + exposeHeaders - The allowlisted headers, which clients are allowed to access
# + allowCredentials - Specifies whether credentials are required to access the service
# + maxAge - The maximum duration to cache the preflight from client side
public type CorsConfig record {|
    string[] allowHeaders = [];
    string[] allowMethods = [];
    string[] allowOrigins = [];
    string[] exposeHeaders = [];
    boolean allowCredentials = false;
    decimal maxAge= -1;
|};

# The annotation which is used to configure an HTTP service.
public annotation HttpServiceConfig ServiceConfig on service, type;

# Configuration for an HTTP resource.
# The `auth` field of jBallerina's `HttpResourceConfig` is not supported in this implementation.
#
# + name - The name of the resource
# + consumes - The media types which are accepted by resource
# + produces - The media types which are produced by resource
# + cors - The cross origin resource sharing configurations for the resource. If not set, the resource will inherit the CORS behaviour of the enclosing service.
# + transactionInfectable - Allow to participate in the distributed transactions if value is true
# + linkedTo - The array of linked resources
public type HttpResourceConfig record {|
    string name?;
    string[] consumes = [];
    string[] produces = [];
    CorsConfig cors = {};
    boolean transactionInfectable = true;
    LinkedTo[] linkedTo?;
|};

# The annotation which is used to configure an HTTP resource.
public annotation HttpResourceConfig ResourceConfig on object function;

# A record for providing configurations for content compression.
public type CompressionConfig record {|
    # The status of compression
    Compression enable = COMPRESSION_AUTO;
    # Content types which are allowed for compression
    string[] contentTypes = [];
|};

public type LinkedTo record {|
    # Name of the linked resource
    string name;
    # Name of the relationship between the linked resource and the current resource.
    # Defaulted to the IANA link relation `self`
    string relation = "self";
    # Method allowed in the linked resource
    string method?;
|};

# Defines the possible values for the chunking configuration in HTTP services and clients.
#
# `AUTO`: If the payload is less than 8KB, content-length header is set in the outbound request/response,
#         otherwise chunking header is set in the outbound request/response
# `ALWAYS`: Always set chunking header in the response
# `NEVER`: Never set the chunking header even if the payload is larger than 8KB in the outbound request/response
public type Chunking CHUNKING_AUTO|CHUNKING_ALWAYS|CHUNKING_NEVER;

# If the payload is less than 8KB, content-length header is set in the outbound request/response,
# otherwise chunking header is set in the outbound request/response.}
public const CHUNKING_AUTO = "AUTO";

# Always set chunking header in the response.
public const CHUNKING_ALWAYS = "ALWAYS";

# Never set the chunking header even if the payload is larger than 8KB in the outbound request/response.
public const CHUNKING_NEVER = "NEVER";
