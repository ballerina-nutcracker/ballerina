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

import ballerina/crypto;
import ballerina/io;
import ballerina/jwt;

const RSA_KEY = string `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDybuZlJ0QI18md
JSc9kw02U0sWe/JmJAbrkONM+qkrzesQH9JhfIgW5RYl3dIp1wDSPb+QZWsTOn5a
9zYoVEqyhnw4T7veUiXng7myD65+jFSh26dvqDPBDpwfbVrLnu7yaeKCez9fRVUS
F4AXTI5uS2e2G1LPTy/dHiZrk8HHv0bDme7NWIq2lJSe+Qzix3X9K3s9BEq19xYz
6lKFVXVdJljpU1rPypmZb+vt8pDViF05DcwN3XGX6OhqgufrqEz6LaG3oWwAVFhM
SGLCrReONJxr+2XwjAV7CbMi6kqJg/8rleDOqgWESfEsoy2sA0rdprS069K25k/c
ypYiEwdTAgMBAAECggEAQJ/rbr9+1VSge/Fxemp+tSKS+5Vi0nN8HMNstBKFkhW8
OXoiHLaTmHbM5p8w7NJRd7naHOXMGz4WfK01lJ6Tou9Sobgvfl4rmIF2uM6RV06V
KOhiw9L0qNv6Ji6Ca5EcZSZwqROtmvniaYY2MP/JYAFdzh+OfZDAU5M77qVenJ4N
qwNwAL9/t5/gjSt8Nuy4Njiqj7XyQ2kZ5nm4rAy9ehnH6yYM2EPJhUUoViDaBV0n
AZRgAENTqJNYSLK63zuMzWYZEDv2Gv3r9wbGlodIpp9LNVc4x6PTFWtpmT2phClJ
kMGQYuHMGbk3LpxU4ECNd5ZwxNorORh/1wnLkn374QKBgQD9MPibc9BwSexnY015
Yq/68Ckg3m7si3cOs+4uW9DdmiwW1S+VHNKIa7GBHKayea+q36AEg0Pk884GzO0i
MeiAGHSJ3YX76HhBXH5nQjCRD0Yz6LeGEo2tjafN75GkAPxEwf1XFr+zg+PqbZDi
/eXhAo7eqvJjqEOq7NoH88xvGwKBgQD1H2CZmMjHHcv1W51fATxVLADnzScdJHFb
PMJjjrIVtmLinGPlnDeb0QFccCd+9cftbBtQi0iiAe67Vuj7edLZf9K6+NXqcAQU
hQLnLDrFiZIXrHcLHLtWEAu3eLLJ9/HxL/YDA+Ca7OPM5Wmin3+27qR6HLAii96u
cIu6osd0KQKBgQDKxVt08+2s4CcqLccwPa7+YzQtFLb1TkRNTCCmk/XUY9KHfria
zPh3NPh1lIKUUQw05t96fb30MKWBoMERL7MqU2uRtg3Fz0nv3qmE/Od0ZS0uCWRh
RZ6dreMD9KdJF1jxoALKqndgx3HSn2j3watG5wuQnIt+8JUw4HtE/yVsnQKBgQC4
20DulNpVz5LRJpY336awVM8aAQ1OXP1f4SJ4ElUbKsDIRUjREE6fo69zFuGyTYjI
9I5k36WXB/4SKijp/aWVgWBawO4SLwfksarOy/xGnMkgMWBVcQWWkeNEKjIrPmGn
kpPlEjeAnsXs79YKXl7ihnZj4N60Ludwhk/aIIhqgQKBgCmKwtnKNcOCvP+W+M9F
4okhR3XxphGXgL5nsTN4ztNw7Q/L6AtxM1dJN2dvDr702xDvtpwzJlsr7wimfK7D
Lv/qune7RNkjuBvi8HJTMQMZI6V6LoPtT2GHoYLeF4GiyjiMorsvBpYjrOmDFBtz
d5ubSxoSnzyCBtyAZZiAYNKg
-----END PRIVATE KEY-----`;

const RSA_CERT = string `-----BEGIN CERTIFICATE-----
MIIDCTCCAfGgAwIBAgIUNGFCKHaRv3N48DcAqnHSPoxV+QgwDQYJKoZIhvcNAQEL
BQAwEzERMA8GA1UEAwwIand0LXRlc3QwIBcNMjYxMTE2MDUzMzI5WhgPMjEyNjEw
MjMwNTMzMjlaMBMxETAPBgNVBAMMCGp3dC10ZXN0MIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEA8m7mZSdECNfJnSUnPZMNNlNLFnvyZiQG65DjTPqpK83r
EB/SYXyIFuUWJd3SKdcA0j2/kGVrEzp+Wvc2KFRKsoZ8OE+73lIl54O5sg+ufoxU
odunb6gzwQ6cH21ay57u8mnigns/X0VVEheAF0yObktnthtSz08v3R4ma5PBx79G
w5nuzViKtpSUnvkM4sd1/St7PQRKtfcWM+pShVV1XSZY6VNaz8qZmW/r7fKQ1Yhd
OQ3MDd1xl+joaoLn66hM+i2ht6FsAFRYTEhiwq0XjjSca/tl8IwFewmzIupKiYP/
K5XgzqoFhEnxLKMtrANK3aa0tOvStuZP3MqWIhMHUwIDAQABo1MwUTAdBgNVHQ4E
FgQUYe4BplhibNykgIES+l3cGGrkamIwHwYDVR0jBBgwFoAUYe4BplhibNykgIES
+l3cGGrkamIwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAUeHs
UaWCx7IisyC5PyoHrMVW35dgBJxl+hJ9kuWbI3JzXfSL8Wo7lITvBAv1OrNtzyt3
JR0AC3e4XnB3J+8N0oqfn/jRnCnx34UfMrVpbWNRDxnTti5tha9YFJRBTGkCvA9y
50x0b5sbqunYHBmKZuMWDTM7gHyb84OfcgIFGjML8m7bAwvwlrJGaUWTRt/u/vZP
qX5BKZ46vBsp0hwoGK1tuHSUXoJdk9NrNExWrDvQzV+a51okkC03cr90+fb9pHNI
GV45iANuJHIJGAwok8DMfVrjFex7eCaGp2W7pmJPE4n291PlqiHPqSfC5VokeRIp
lNtaDeH7sYA0ZcDWfg==
-----END CERTIFICATE-----`;

const RSA_KEY_PATH = "/tmp/bal_jwt_validate_rsa_key.pem";
const RSA_CERT_PATH = "/tmp/bal_jwt_validate_rsa_cert.pem";

function show(jwt:Payload|jwt:Error result) {
    if result is jwt:Error {
        io:println(result.message(), " (cause: ", result.cause() is error, ")");
    } else {
        io:println("valid ", result?.sub ?: "-");
    }
}

public function main() returns error? {
    check io:fileWriteString(RSA_KEY_PATH, RSA_KEY);
    check io:fileWriteString(RSA_CERT_PATH, RSA_CERT);

    string token = check jwt:issue({
        issuer: "wso2",
        username: "alice",
        audience: ["a", "b"],
        jwtId: "j1",
        keyId: "k1",
        customClaims: {"scp": "read", "n": 1, "o": {"x": [1]}},
        signatureConfig: {algorithm: jwt:HS256, config: "secret"}
    });
    show(jwt:validate(token, {
        issuer: "wso2",
        username: "alice",
        audience: "b",
        jwtId: "j1",
        keyId: "k1",
        customClaims: {"scp": "read", "n": 1, "o": {"x": [1]}},
        clockSkew: 30,
        signatureConfig: {secret: "secret"}
    })); // @output valid alice
    show(jwt:validate(token, {audience: ["c", "a"]})); // @output valid alice
    // Custom claims are compared by their string form, as in jBallerina.
    show(jwt:validate(token, {customClaims: {"n": "1"}})); // @output valid alice
    show(jwt:validate(token, {signatureConfig: {secret: "other"}})); // @output JWT signature validation with shared secret has failed. (cause: false)
    show(jwt:validate(token, {issuer: "x"})); // @output JWT contained invalid issuer name 'wso2' (cause: false)
    show(jwt:validate(token, {username: "x"})); // @output JWT contained invalid username 'alice' (cause: false)
    show(jwt:validate(token, {audience: "c"})); // @output JWT contained invalid audience. (cause: false)
    show(jwt:validate(token, {audience: ["c", "d"]})); // @output JWT contained invalid audience. (cause: false)
    show(jwt:validate(token, {jwtId: "x"})); // @output JWT contained invalid JWT ID 'j1' (cause: false)
    show(jwt:validate(token, {keyId: "x"})); // @output JWT contained invalid key ID 'k1' (cause: false)
    show(jwt:validate(token, {customClaims: {"zz": 1}})); // @output JWT must contain a 'zz' custom claim. (cause: false)
    show(jwt:validate(token, {customClaims: {"scp": "write"}})); // @output JWT contained invalid custom claim 'scp: read' (cause: false)

    string bare = check jwt:issue({audience: "single", signatureConfig: {algorithm: jwt:HS384, config: "secret"}});
    show(jwt:validate(bare, {issuer: "x"})); // @output JWT must contain a valid issuer name. (cause: false)
    show(jwt:validate(bare, {username: "x"})); // @output JWT must contain a valid username. (cause: false)
    show(jwt:validate(bare, {jwtId: "x"})); // @output JWT must contain a valid JWT ID. (cause: false)
    show(jwt:validate(bare, {keyId: "x"})); // @output JWT must contain a valid key ID. (cause: false)
    show(jwt:validate(bare, {audience: ["x", "single"], signatureConfig: {secret: "secret"}})); // @output valid -
    show(jwt:validate(bare, {audience: "other"})); // @output JWT contained invalid audience. (cause: false)
    string noAudience = check jwt:issue({signatureConfig: {algorithm: jwt:HS512, config: "secret"}});
    show(jwt:validate(noAudience, {audience: "x"})); // @output JWT must contain a valid audience. (cause: false)
    show(jwt:validate(noAudience, {signatureConfig: {secret: "secret"}})); // @output valid -

    show(jwt:validate("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhbGljZSIsImV4cCI6MTAwMH0.sig", {})); // @output JWT is expired. (cause: false)
    show(jwt:validate("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhbGljZSIsImV4cCI6MTAwMH0.sig", {clockSkew: 100000000000})); // @output valid alice
    show(jwt:validate("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhbGljZSIsIm5iZiI6OTk5OTk5OTk5OTl9.sig", {})); // @output JWT is used before not-before-time (nbf). (cause: false)
    show(jwt:validate("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhbGljZSIsIm5iZiI6OTk5OTk5OTk5OTl9.sig", {clockSkew: 100000000000})); // @output valid alice
    string expiring = check jwt:issue({username: "alice", expTime: -10, signatureConfig: {algorithm: jwt:HS256, config: "secret"}});
    show(jwt:validate(expiring, {})); // @output JWT is expired. (cause: false)
    show(jwt:validate(expiring, {clockSkew: 60})); // @output valid alice

    // jBallerina panics with IndexOutOfRange here: https://github.com/ballerina-platform/ballerina-library/issues/9710
    show(jwt:validate("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhbGljZSJ9", {})); // @output valid alice
    show(jwt:validate("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhbGljZSJ9", {signatureConfig: {secret: "secret"}})); // @output Not a valid JWS. Signature part is required. (cause: false)
    show(jwt:validate("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhbGljZSJ9.a$b", {signatureConfig: {secret: "secret"}})); // @output Base64 URL decode failed for JWT signature. (cause: true)
    show(jwt:validate(token, {signatureConfig: {}})); // @output valid alice
    show(jwt:validate(token, {signatureConfig: {certFile: RSA_CERT_PATH}})); // @output Unsupported RSA algorithm 'HS256'. (cause: false)
    show(jwt:validate(bare, {signatureConfig: {jwksConfig: {url: "https://localhost/jwks"}}})); // @output Key ID (kid) is not provided in JOSE header. (cause: false)
    show(jwt:validate("a.b.c.d", {})); // @output Invalid JWT. (cause: false)

    string rs256 = check jwt:issue({username: "bob", signatureConfig: {config: {keyFile: RSA_KEY_PATH}}});
    show(jwt:validate(rs256, {signatureConfig: {certFile: RSA_CERT_PATH}})); // @output valid bob
    crypto:PublicKey publicKey = check crypto:decodeRsaPublicKeyFromCertFile(RSA_CERT_PATH);
    show(jwt:validate(rs256, {signatureConfig: {certFile: publicKey}})); // @output valid bob
    // jBallerina rejects a key without a certificate as expired: https://github.com/ballerina-platform/ballerina-library/issues/9711
    // A key without a certificate now reaches the signature check; this record carries no key material, so it fails there.
    crypto:PublicKey keyWithoutCertificate = {algorithm: crypto:RSA};
    show(jwt:validate(rs256, {signatureConfig: {certFile: keyWithoutCertificate}})); // @output SHA256 signature verification failed. (cause: true)
    show(jwt:validate(rs256, {signatureConfig: {secret: "secret"}})); // @output Unsupported HMAC algorithm 'RS256'. (cause: false)
    show(jwt:validate(rs256, {signatureConfig: {certFile: "/tmp/bal_jwt_missing_cert.pem"}})); // @output Failed to decode public key. (cause: true)
    string rs384 = check jwt:issue({signatureConfig: {algorithm: jwt:RS384, config: {keyFile: RSA_KEY_PATH}}});
    show(jwt:validate(rs384, {signatureConfig: {certFile: RSA_CERT_PATH}})); // @output valid -
    string rs512 = check jwt:issue({signatureConfig: {algorithm: jwt:RS512, config: {keyFile: RSA_KEY_PATH}}});
    show(jwt:validate(rs512, {signatureConfig: {certFile: RSA_CERT_PATH}})); // @output valid -
    string hs384Tampered = check jwt:issue({signatureConfig: {algorithm: jwt:HS384, config: "secret"}});
    show(jwt:validate(hs384Tampered, {signatureConfig: {secret: "other"}})); // @output JWT signature validation with shared secret has failed. (cause: false)
    string hs512Tampered = check jwt:issue({signatureConfig: {algorithm: jwt:HS512, config: "secret"}});
    show(jwt:validate(hs512Tampered, {signatureConfig: {secret: "other"}})); // @output JWT signature validation with shared secret has failed. (cause: false)
}
