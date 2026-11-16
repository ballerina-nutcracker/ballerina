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

import ballerina/cache;
import ballerina/log;
import ballerina/time;

# Represents the listener JWT Auth provider, which is used to authenticate the provided credentials (JWT) against
# the provided JWT validator configurations.
# ```ballerina
# jwt:ListenerJwtAuthProvider provider = new({
#     issuer: "example",
#     audience: "ballerina",
#     signatureConfig: {
#         certificateAlias: "ballerina",
#         trustStore: {
#             path: "/path/to/truststore.p12",
#             password: "ballerina"
#         }
#     }
# });
# ```
public isolated class ListenerJwtAuthProvider {

    private final ValidatorConfig validatorConfig;
    private final cache:Cache? jwtCache;

    # Provides authentication based on the provided JWT.
    #
    # + validatorConfig - JWT validator configurations
    public isolated function init(ValidatorConfig validatorConfig) {
        self.validatorConfig = validatorConfig.clone();
        cache:CacheConfig? jwtCacheConfig = validatorConfig?.cacheConfig;
        if jwtCacheConfig is cache:CacheConfig {
            self.jwtCache = new(jwtCacheConfig);
        } else {
            self.jwtCache = ();
        }
    }

    # Authenticates the provided JWT against the configured validator.
    #```ballerina
    # boolean result = check provider.authenticate("<credential>");
    # ```
    #
    # + credential - JWT to be authenticated
    # + return - `jwt:Payload` if authentication is successful or else a `jwt:Error` if an error occurred
    public isolated function authenticate(string credential) returns Payload|Error {
        string[] jwtComponents = splitByDot(credential);
        if jwtComponents.length() != 3 {
            return prepareError("Credential format does not match to JWT format.");
        }

        cache:Cache? jwtCache = self.getJwtCache();
        if jwtCache is cache:Cache && jwtCache.hasKey(credential) {
            Payload? payload = validateFromCache(jwtCache, credential);
            if payload is Payload {
                return payload;
            }
        }

        Payload|Error validationResult = validateJwt(credential, self.getValidatorConfig());
        if validationResult is Payload {
            if jwtCache is cache:Cache {
                addToCache(jwtCache, credential, validationResult);
            }
            return validationResult;
        } else {
            return prepareError("JWT validation failed.", validationResult);
        }
    }

    // A final isolated-object field still needs a lock here (#1431).
    private isolated function getJwtCache() returns cache:Cache? {
        lock {
            return self.jwtCache;
        }
    }

    private isolated function getValidatorConfig() returns ValidatorConfig {
        lock {
            return self.validatorConfig.clone();
        }
    }
}

isolated function validateFromCache(cache:Cache jwtCache, string jwt) returns Payload? {
    any|cache:Error cachedResult = jwtCache.get(jwt);
    if cachedResult is any {
        Payload payload = <Payload> cachedResult;
        int? expTime = payload?.exp;
        // convert to current time and check the expiry time
        [int, decimal] currentTime = time:utcNow();
        if expTime is () || expTime > currentTime[0] {
            return payload;
        }
        cache:Error? result = jwtCache.invalidate(jwt);
        if result is cache:Error {
            log:printDebug("Failed to invalidate JWT from the cache.", 'error = result);
        }
    } else {
        log:printDebug("Failed to retrieve JWT entry from the cache.", 'error = cachedResult);
    }
    return;
}

isolated function addToCache(cache:Cache jwtCache, string jwt, Payload payload) {
    cache:Error? result = jwtCache.put(jwt, payload);
    if result is cache:Error {
        log:printDebug("Failed to add JWT to the cache.", 'error = result);
    }
}
