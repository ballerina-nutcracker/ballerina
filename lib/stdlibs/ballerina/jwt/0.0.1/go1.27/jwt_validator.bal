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

# Decodes the provided JWT into the header and payload.
# ```ballerina
# [jwt:Header, jwt:Payload] [header, payload] = check jwt:decode(jwt);
# ```
#
# + jwt - JWT that needs to be decoded
# + return - The `jwt:Header` and `jwt:Payload` as a tuple or else a `jwt:Error` if an error occurred
public isolated function decode(string jwt) returns [Header, Payload]|Error {
    string[] encodedJwtComponents = check getJwtComponents(jwt);
    Header header = check getHeader(encodedJwtComponents[0]);
    Payload payload = check getPayload(encodedJwtComponents[1]);
    return [header, payload];
}

isolated function getJwtComponents(string jwt) returns string[]|Error {
    string[] jwtComponents = splitByDot(jwt);
    if jwtComponents.length() < 2 || jwtComponents.length() > 3 {
        return prepareError("Invalid JWT.");
    }
    return jwtComponents;
}

// Splits as jBallerina's re `\.`.split does, keeping empty segments; lang.regexp is not available yet.
isolated function splitByDot(string value) returns string[] {
    string[] components = [];
    int startIndex = 0;
    int? dotIndex = value.indexOf(".");
    while dotIndex is int {
        components.push(value.substring(startIndex, dotIndex));
        startIndex = dotIndex + 1;
        dotIndex = value.indexOf(".", startIndex);
    }
    components.push(value.substring(startIndex));
    return components;
}

isolated function getHeader(string encodedHeader) returns Header|Error {
    byte[]|Error decodedHeader = decodeBase64Url(encodedHeader);
    if decodedHeader is byte[] {
        string|error result = string:fromBytes(decodedHeader);
        if result is string {
            json|error jsonHeader = fromJsonString(result);
            if jsonHeader is map<json> {
                return parseHeader(jsonHeader);
            } else if jsonHeader is json {
                return prepareError("JWT header is not a JSON object.");
            } else {
                return prepareError("String to JSON conversion failed for JWT header.", jsonHeader);
            }
        } else {
            return prepareError("Failed to convert byte[] of decoded header to string.", result);
        }
    } else {
        return prepareError("Base64 URL decode failed for JWT header.", decodedHeader);
    }
}

isolated function getPayload(string encodedPayload) returns Payload|Error {
    byte[]|Error decodedPayload = decodeBase64Url(encodedPayload);
    if decodedPayload is byte[] {
        string|error result = string:fromBytes(decodedPayload);
        if result is string {
            json|error jsonPayload = fromJsonString(result);
            if jsonPayload is map<json> {
                return parsePayload(jsonPayload);
            } else if jsonPayload is json {
                return prepareError("JWT payload is not a JSON object.");
            } else {
                return prepareError("String to JSON conversion failed for JWT paylaod.", jsonPayload);
            }
        } else {
            return prepareError("Failed to convert byte[] of decoded payload to string.", result);
        }
    } else {
        return prepareError("Base64 URL decode failed for JWT payload.", decodedPayload);
    }
}

isolated function parseHeader(map<json> headerMap) returns Header|Error {
    Header header = {};
    string[] keys = headerMap.keys();
    foreach string key in keys {
        match key {
            ALG => {
                if headerMap[key] == "RS256" {
                    header.alg = RS256;
                } else if headerMap[key] == "RS384" {
                    header.alg = RS384;
                } else if headerMap[key] == "RS512" {
                    header.alg = RS512;
                } else if headerMap[key] == "HS256" {
                    header.alg = HS256;
                } else if headerMap[key] == "HS384" {
                    header.alg = HS384;
                } else if headerMap[key] == "HS512" {
                    header.alg = HS512;
                } else {
                    return prepareError("Unsupported signing algorithm '" + headerMap[key].toString() + "'.");
                }
            }
            TYP => {
                header.typ = check stringField(headerMap, key, "header");
            }
            CTY => {
                header.cty = check stringField(headerMap, key, "header");
            }
            KID => {
                header.kid = check stringField(headerMap, key, "header");
            }
            _ => {
                header[key] = headerMap[key];
            }
        }
    }
    return header;
}

isolated function parsePayload(map<json> payloadMap) returns Payload|Error {
    Payload payload = {};
    string[] keys = payloadMap.keys();
    foreach string key in keys {
        match key {
            ISS => {
                payload.iss = check stringField(payloadMap, key, "payload");
            }
            SUB => {
                payload.sub = check stringField(payloadMap, key, "payload");
            }
            AUD => {
                json aud = payloadMap[key];
                payload.aud = aud is json[] ? check convertToStringArray(aud) : check stringField(payloadMap, key, "payload");
            }
            EXP => {
                payload.exp = timeClaim(payloadMap[key]);
            }
            NBF => {
                payload.nbf = timeClaim(payloadMap[key]);
            }
            IAT => {
                payload.iat = timeClaim(payloadMap[key]);
            }
            JTI => {
                payload.jti = check stringField(payloadMap, key, "payload");
            }
            _ => {
                payload[key] = payloadMap[key];
            }
        }
    }
    return payload;
}

// jBallerina casts these claims with `<string>` and panics on other JSON types: https://github.com/ballerina-platform/ballerina-library/issues/9709
isolated function stringField(map<json> claims, string key, string part) returns string|Error {
    json value = claims[key];
    if value is string {
        return value;
    }
    return prepareError("Invalid value for the JWT " + part + " field '" + key + "'. A string is expected.");
}

isolated function timeClaim(json value) returns int {
    int|error result = int:fromString(value.toString());
    return result is int ? result : 0;
}

isolated function validateMandatoryHeaderFields(Header header) returns boolean {
    SigningAlgorithm? alg = header?.alg;
    return alg is SigningAlgorithm;
}

isolated function convertToStringArray(json[] jsonData) returns string[]|Error {
    string[] values = [];
    int i = 0;
    foreach json jsonVal in jsonData {
        values[i] = jsonVal.toString();
        i = i + 1;
    }
    return values;
}
