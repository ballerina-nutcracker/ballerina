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
import ballerina/crypto;
import ballerina/time;

# Represents JWT validator configurations.
#
# + issuer - Expected issuer, which is mapped to the `iss`
# + username - Expected username, which is mapped to the `sub`
# + audience - Expected audience, which is mapped to the `aud`
# + jwtId - Expected JWT ID, which is mapped to the `jti`
# + keyId - Expected JWT key ID, which is mapped the `kid`
# + customClaims - Expected map of custom claims
# + clockSkew - Clock skew (in seconds) that can be used to avoid token validation failures due to clock synchronization problems
# + signatureConfig - JWT signature configurations
# + cacheConfig - Configurations related to the cache, which are used to store parsed JWT information
public type ValidatorConfig record {
    string issuer?;
    string username?;
    string|string[] audience?;
    string jwtId?;
    string keyId?;
    map<json> customClaims?;
    decimal clockSkew = 0;
    ValidatorSignatureConfig signatureConfig?;
    cache:CacheConfig cacheConfig?;
};

# Represents JWT signature configurations.
#
# + jwksConfig - JWKS configurations
# + certFile - Public certificate file path or a `crypto:PublicKey`
# + trustStoreConfig - JWT TrustStore configurations
# + secret - HMAC secret configuration
public type ValidatorSignatureConfig record {|
    record {|
        string url;
        cache:CacheConfig cacheConfig?;
        ClientConfiguration clientConfig = {};
    |} jwksConfig?;
    string|crypto:PublicKey certFile?;
    record {|
        crypto:TrustStore trustStore;
        string certAlias;
    |} trustStoreConfig?;
    string secret?;
|};

# Represents the configurations of the client used to call the JWKS endpoint.
#
# + httpVersion - The HTTP version of the client
# + secureSocket - SSL/TLS-related configurations
public type ClientConfiguration record {|
    HttpVersion httpVersion = HTTP_1_1;
    SecureSocket secureSocket?;
|};

# Represents the HTTP versions.
public enum HttpVersion {
    HTTP_1_1,
    HTTP_2
}

# Represents the SSL/TLS configurations.
#
# + disable - Disable SSL validation
# + cert - Configurations associated with the `crypto:TrustStore` or single certificate file that the client trusts
# + key - Configurations associated with the `crypto:KeyStore` or combination of certificate and private key of the client
public type SecureSocket record {|
    boolean disable = false;
    crypto:TrustStore|string cert?;
    crypto:KeyStore|CertKey key?;
|};

# Represents the combination of the certificate file path, private key file path, and private key password if encrypted.
#
# + certFile - A file containing the certificate
# + keyFile - A file containing the private key
# + keyPassword - Password of the private key (if encrypted)
public type CertKey record {|
   string certFile;
   string keyFile;
   string keyPassword?;
|};

# Validates the provided JWT, against the provided configurations.
#```ballerina
# jwt:Payload result = check jwt:validate(jwt, validatorConfig);
# ```
#
# + jwt - JWT that needs to be validated
# + validatorConfig - JWT validator configurations
# + return - `jwt:Payload` or else a `jwt:Error` if an error occurred
public isolated function validate(string jwt, ValidatorConfig validatorConfig) returns Payload|Error {
    return validateJwt(jwt, validatorConfig);
}

isolated function validateJwt(string jwt, ValidatorConfig validatorConfig) returns Payload|Error {
    [Header, Payload] decoded = check decode(jwt);
    Header header = decoded[0];
    Payload payload = decoded[1];
    if !validateMandatoryHeaderFields(header) {
        return prepareError("Mandatory field signing algorithm (alg) is not provided in JOSE header.");
    }
    check validateJwtRecords(header, payload, validatorConfig);
    check validateSignature(jwt, header, validatorConfig);
    return payload;
}

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

isolated function getJwtSignature(string encodedSignature) returns byte[]|Error {
    byte[]|Error signature = decodeBase64Url(encodedSignature);
    if signature is byte[] {
        return signature;
    } else {
        return prepareError("Base64 URL decode failed for JWT signature.", signature);
    }
}

isolated function validateSignature(string jwt, Header header, ValidatorConfig validatorConfig) returns Error? {
    SigningAlgorithm alg = <SigningAlgorithm>header?.alg;  // The `()` value is already validated.
    ValidatorSignatureConfig? validatorSignatureConfig = validatorConfig?.signatureConfig;
    // jBallerina goes on to read a missing signature segment and panics: https://github.com/ballerina-platform/ballerina-library/issues/9710
    if validatorSignatureConfig is () {
        return;
    }
    if alg == NONE {
        return prepareError("Not a valid JWS. Signing algorithm is 'NONE'.");
    }
    string[] encodedJwtComponents = check getJwtComponents(jwt);
    if encodedJwtComponents.length() == 2 {
        return prepareError("Not a valid JWS. Signature part is required.");
    }
    string headerPayloadPart = encodedJwtComponents[0] + "." + encodedJwtComponents[1];
    byte[] assertion = headerPayloadPart.toBytes();
    byte[] signature = check getJwtSignature(encodedJwtComponents[2]);
    var jwksConfig = validatorSignatureConfig?.jwksConfig;
    var certFile = validatorSignatureConfig?.certFile;
    var trustStoreConfig = validatorSignatureConfig?.trustStoreConfig;
    string? secret = validatorSignatureConfig?.secret;
    if jwksConfig !is () {
        string? kid = header?.kid;
        if kid is string {
            return prepareError("JWKS-based signature validation is not yet supported.");
        } else {
            return prepareError("Key ID (kid) is not provided in JOSE header.");
        }
    } else if certFile !is () {
        // Type tests against crypto:PublicKey fail for keys from the crypto natives (#1676), so test the other cases.
        crypto:PublicKey|crypto:Error publicKey = certFile is string ? crypto:decodeRsaPublicKeyFromCertFile(certFile)
            : certFile;
        if publicKey is crypto:Error {
            return prepareError("Failed to decode public key.", publicKey);
        }
        if !validateCertificate(publicKey) {
            return prepareError("Public key certificate validity period has passed.");
        }
        boolean signatureValidation = check assertRsaSignature(alg, assertion, signature, publicKey);
        if !signatureValidation {
            return prepareError("JWT signature validation with public key configurations has failed.");
        }
    } else if trustStoreConfig !is () {
        crypto:PublicKey|crypto:Error publicKey = crypto:decodeRsaPublicKeyFromTrustStore(trustStoreConfig.trustStore,
            trustStoreConfig.certAlias);
        if publicKey is crypto:Error {
            return prepareError("Failed to decode public key.", publicKey);
        }
        if !validateCertificate(publicKey) {
            return prepareError("Public key certificate validity period has passed.");
        }
        boolean signatureValidation = check assertRsaSignature(alg, assertion, signature, publicKey);
        if !signatureValidation {
            return prepareError("JWT signature validation with TrustStore configurations has failed.");
        }
    } else if secret !is () {
        boolean signatureValidation = check assertHmacSignature(alg, assertion, signature, secret);
        if !signatureValidation {
            return prepareError("JWT signature validation with shared secret has failed.");
        }
    }
}

isolated function validateJwtRecords(Header header, Payload payload, ValidatorConfig validatorConfig) returns Error? {
    string? sub = validatorConfig?.username;
    if sub is string {
        check validateUsername(payload, sub);
    }
    string? iss = validatorConfig?.issuer;
    if iss is string {
        check validateIssuer(payload, iss);
    }
    string|string[]? aud = validatorConfig?.audience;
    if aud is string || aud is string[] {
        check validateAudience(payload, aud);
    }
    string? jwtId = validatorConfig?.jwtId;
    if jwtId is string {
        check validateJwtId(payload, jwtId);
    }
    string? keyId = validatorConfig?.keyId;
    if keyId is string {
        check validateKeyId(header, keyId);
    }
    map<json>? customClaims = validatorConfig?.customClaims;
    if customClaims is map<json> {
        check validateCustomClaims(payload, customClaims);
    }
    int? exp = payload?.exp;
    if exp is int {
        if !validateExpirationTime(exp, <int> validatorConfig.clockSkew) {
            return prepareError("JWT is expired.");
        }
    }
    int? nbf = payload?.nbf;
    if nbf is int {
        if !validateNotBeforeTime(nbf, <int> validatorConfig.clockSkew) {
            return prepareError("JWT is used before not-before-time (nbf).");
        }
    }
    return;
}

// A key without a certificate has no validity period to check. jBallerina rejects such a key as expired:
// https://github.com/ballerina-platform/ballerina-library/issues/9711
isolated function validateCertificate(crypto:PublicKey publicKey) returns boolean {
    crypto:Certificate? certificate = publicKey?.certificate;
    if certificate is () {
        return true;
    }
    [int, decimal] currentTime = time:utcNow();
    [int, decimal] notBefore = certificate.notBefore;
    [int, decimal] notAfter = certificate.notAfter;
    return currentTime[0] >= notBefore[0] && currentTime[0] <= notAfter[0];
}

isolated function assertRsaSignature(SigningAlgorithm alg, byte[] assertion, byte[] signaturePart,
                                     crypto:PublicKey publicKey) returns boolean|Error {
    match alg {
        RS256 => {
            boolean|crypto:Error result = crypto:verifyRsaSha256Signature(assertion, signaturePart, publicKey);
            if result is boolean {
                return result;
            } else {
                return prepareError("SHA256 signature verification failed.", result);
            }
        }
        RS384 => {
            boolean|crypto:Error result = crypto:verifyRsaSha384Signature(assertion, signaturePart, publicKey);
            if result is boolean {
                return result;
            } else {
                return prepareError("SHA384 signature verification failed.", result);
            }
        }
        RS512 => {
            boolean|crypto:Error result = crypto:verifyRsaSha512Signature(assertion, signaturePart, publicKey);
            if result is boolean {
                return result;
            } else {
                return prepareError("SHA512 signature verification failed.", result);
            }
        }
    }
    return prepareError("Unsupported RSA algorithm '" + alg.toString() + "'.");
}

isolated function assertHmacSignature(SigningAlgorithm alg, byte[] assertion, byte[] signaturePart,
                                      string secret) returns boolean|Error {
    match alg {
        HS256 => {
            byte[]|crypto:Error signature = crypto:hmacSha256(assertion, secret.toBytes());
            if signature is byte[] {
                return signature == signaturePart;
            } else {
                return prepareError("HMAC secret key validation failed for SHA256 algorithm.", signature);
            }
        }
        HS384 => {
            byte[]|crypto:Error signature = crypto:hmacSha384(assertion, secret.toBytes());
            if signature is byte[] {
                return signature == signaturePart;
            } else {
                return prepareError("HMAC secret key validation failed for SHA384 algorithm.", signature);
            }
        }
        HS512 => {
            byte[]|crypto:Error signature = crypto:hmacSha512(assertion, secret.toBytes());
            if signature is byte[] {
                return signature == signaturePart;
            } else {
                return prepareError("HMAC secret key validation failed for SHA512 algorithm.", signature);
            }
        }
    }
    return prepareError("Unsupported HMAC algorithm '" + alg.toString() + "'.");
}

isolated function validateUsername(Payload payload, string usernameConfig) returns Error? {
    string? usernamePayload = payload?.sub;
    if usernamePayload is string {
        if usernamePayload != usernameConfig {
            return prepareError("JWT contained invalid username '" + usernamePayload + "'");
        }
        return;
    } else {
        return prepareError("JWT must contain a valid username.");
    }
}

isolated function validateIssuer(Payload payload, string issuerConfig) returns Error? {
    string? issuePayload = payload?.iss;
    if issuePayload is string {
        if issuePayload != issuerConfig {
            return prepareError("JWT contained invalid issuer name '" + issuePayload + "'");
        }
        return;
    } else {
        return prepareError("JWT must contain a valid issuer name.");
    }
}

isolated function validateAudience(Payload payload, string|string[] audienceConfig) returns Error? {
    string|string[]? audiencePayload = payload?.aud;
    if audiencePayload is string {
        if audienceConfig is string {
            if audiencePayload == audienceConfig {
                return;
            }
        } else {
            foreach string audience in audienceConfig {
                if audience == audiencePayload {
                    return;
                }
            }
        }
        return prepareError("JWT contained invalid audience.");
    } else if audiencePayload is string[] {
        if audienceConfig is string {
            foreach string audience in audiencePayload {
                if audience == audienceConfig {
                    return;
                }
            }
        } else {
            foreach string audienceC in audienceConfig {
                foreach string audienceP in audiencePayload {
                    if audienceC == audienceP {
                        return;
                    }
                }
            }
        }
        return prepareError("JWT contained invalid audience.");
    } else {
        return prepareError("JWT must contain a valid audience.");
    }
}

isolated function validateJwtId(Payload payload, string jwtIdConfig) returns Error? {
    string? jwtIdPayload = payload?.jti;
    if jwtIdPayload is string {
        if jwtIdPayload != jwtIdConfig {
            return prepareError("JWT contained invalid JWT ID '" + jwtIdPayload + "'");
        }
        return;
    } else {
        return prepareError("JWT must contain a valid JWT ID.");
    }
}

isolated function validateKeyId(Header header, string keyIdConfig) returns Error? {
    string? keyIdHeader = header?.kid;
    if keyIdHeader is string {
        if keyIdHeader != keyIdConfig {
            return prepareError("JWT contained invalid key ID '" + keyIdHeader + "'");
        }
        return;
    } else {
        return prepareError("JWT must contain a valid key ID.");
    }
}

isolated function validateCustomClaims(Payload payload, map<json> customClaims) returns Error? {
    foreach string key in customClaims.keys() {
        anydata claim = payload[key];
        json customClaimPayload = claim is json ? claim : claim.toString();
        if customClaimPayload is () {
            return prepareError("JWT must contain a '" + key + "' custom claim.");
        }
        json customClaimConfig = customClaims[key];
        if customClaimPayload.toString() != customClaimConfig.toString() {
            return prepareError("JWT contained invalid custom claim '" + key + ": " + customClaimPayload.toString() + "'");
        }
    }
}

isolated function validateExpirationTime(int expTime, int clockSkew) returns boolean {
    [int, decimal] currentTime = time:utcNow();
    if clockSkew > 0 {
        return expTime + clockSkew >= currentTime[0];
    } else {
        return expTime >= currentTime[0];
    }
}

isolated function validateNotBeforeTime(int nbf, int clockSkew) returns boolean {
    [int, decimal] currentTime = time:utcNow();
    if clockSkew > 0 {
        return nbf - clockSkew <= currentTime[0];
    } else {
        return nbf <= currentTime[0];
    }
}
