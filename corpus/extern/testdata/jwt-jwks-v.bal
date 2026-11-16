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

import ballerina/io;
import ballerina/jwt;

function show(jwt:Payload|jwt:Error result) {
    if result is jwt:Error {
        error? cause = result.cause();
        io:println(result.message(), " | ", cause is error ? cause.message() : "no cause");
    } else {
        io:println("valid ", result?.sub ?: "-");
    }
}

function jwks(string path) returns jwt:ValidatorConfig => {signatureConfig: {jwksConfig: {url: "http://testserver" + path}}};

public function main() returns error? {
    string token = check jwt:issue({username: "bob", keyId: "k1", signatureConfig: {config: {keyFile: "testdata/jwt/rsa.key"}}});
    show(jwt:validate(token, jwks("/jwks"))); // @output valid bob
    show(jwt:validate(token, {signatureConfig: {jwksConfig: {url: "http://testserver//jwks"}}})); // @output valid bob
    show(jwt:validate(token, {signatureConfig: {jwksConfig: {url: "testserver/jwks", clientConfig: {httpVersion: jwt:HTTP_2}}}})); // @output valid bob
    string rs512 = check jwt:issue({keyId: "k1", signatureConfig: {algorithm: jwt:RS512, config: {keyFile: "testdata/jwt/rsa.key"}}});
    show(jwt:validate(rs512, jwks("/jwks"))); // @output valid -
    string otherKid = check jwt:issue({keyId: "zz", signatureConfig: {config: {keyFile: "testdata/jwt/rsa.key"}}});
    show(jwt:validate(otherKid, jwks("/jwks"))); // @output No JWK found for kid 'zz'. | no cause
    string wrongKey = check jwt:issue({keyId: "other", signatureConfig: {config: {keyFile: "testdata/jwt/rsa.key"}}});
    show(jwt:validate(wrongKey, jwks("/jwks"))); // @output JWT signature validation with JWKS configurations has failed. | no cause
    string hmac = check jwt:issue({keyId: "k1", signatureConfig: {algorithm: jwt:HS256, config: "secret"}});
    show(jwt:validate(hmac, jwks("/jwks"))); // @output Unsupported RSA algorithm 'HS256'. | no cause

    show(jwt:validate(token, jwks("/created"))); // @output Failed to call JWKS endpoint 'http://testserver/created'. | Failed to get a success response from the endpoint. Response code: '201'. Response body: '{"keys":[]}'
    show(jwt:validate(token, jwks("/missing"))); // @output Failed to call JWKS endpoint 'http://testserver/missing'. | Failed to get a success response from the endpoint. Response code: '404'. Response body: 'missing'
    // jBallerina panics with TypeCastError here: https://github.com/ballerina-platform/ballerina-library/issues/9714
    show(jwt:validate(token, jwks("/not-array"))); // @output Failed to access 'keys' property as an array from the JSON '{"keys":{}}'. | no cause
    show(jwt:validate(token, jwks("/no-keys"))); // @output Failed to access 'keys' property from the JSON '{"x":1}'. | missing map key 'keys'
    show(jwt:validate(token, jwks("/bad-json"))); // @output Failed to convert 'not json' to JSON. | invalid character 'o' in literal null (expecting 'u')
    show(jwt:validate(token, jwks("/no-modulus"))); // @output Failed to access modulus from the JWK '{"kid":"k1","e":"AQAB"}'. | no cause
    show(jwt:validate(token, jwks("/no-exponent"))); // @output Failed to access exponent from the JWK '{"kid":"k1","n":"AQAB"}'. | no cause
    show(jwt:validate(token, jwks("/number-modulus"))); // @output Failed to access modulus or exponent as a 'string' property from the JWK '{"kid":"k1","n":1,"e":"AQAB"}'. | no cause
    show(jwt:validate(token, jwks("/bad-modulus"))); // @output Public key generation failed. | Input is not a valid Base64 URL encoded value. illegal base64 data at input byte 0
    // jBallerina panics with IllegalArgumentException here: https://github.com/ballerina-platform/ballerina-library/issues/9714
    show(jwt:validate(token, jwks("/a b"))); // @output Failed to call JWKS endpoint 'http://testserver/a b'. | Failed to create URI for the provided value "http://testserver/a b".

    jwt:ListenerJwtAuthProvider preloaded = new ({signatureConfig: {jwksConfig: {url: "http://testserver/jwks", cacheConfig: {capacity: 10, evictionFactor: 0.25}}}});
    show(preloaded.authenticate(token)); // @output valid bob
    show(preloaded.authenticate(otherKid)); // @output JWT validation failed. | No JWK found for kid 'zz'.
    jwt:ListenerJwtAuthProvider|error noKid = trap new ({signatureConfig: {jwksConfig: {url: "http://testserver/no-kid", cacheConfig: {capacity: 10, evictionFactor: 0.25}}}});
    io:println(noKid is error ? noKid.message() : "created"); // @output Failed to access 'kid' property from the JSON '{"n":"a","e":"b"}'.
    jwt:ListenerJwtAuthProvider|error numberKid = trap new ({signatureConfig: {jwksConfig: {url: "http://testserver/number-kid", cacheConfig: {capacity: 10, evictionFactor: 0.25}}}});
    io:println(numberKid is error ? numberKid.message() : "created"); // @output Failed to extract 'kid' property as a 'string' from the JSON '{"kid":1,"n":"a","e":"b"}'.
    jwt:ListenerJwtAuthProvider|error unreachable = trap new ({signatureConfig: {jwksConfig: {url: "http://testserver/missing", cacheConfig: {capacity: 10, evictionFactor: 0.25}}}});
    io:println(unreachable is error ? unreachable.message() : "created"); // @output Failed to call JWKS endpoint to preload JWKS to the cache.
    jwt:ListenerJwtAuthProvider withoutPreload = new ({signatureConfig: {jwksConfig: {url: "http://testserver/jwks"}}});
    show(withoutPreload.authenticate(token)); // @output valid bob
}
