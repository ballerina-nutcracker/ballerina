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

const RSA_ENCRYPTED_KEY = string `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIIFLTBXBgkqhkiG9w0BBQ0wSjApBgkqhkiG9w0BBQwwHAQITvL6XaU3bP4CAggA
MAwGCCqGSIb3DQIJBQAwHQYJYIZIAWUDBAEqBBCp+iuI3lWI9LRrgx1Ut9LlBIIE
0P8cWYB1BE/eyudp+rwI3H8YqqP2jY2r0Y8zJ6y7Mk/DYqe2dIhxC0nPCc13DpXy
YI5VfEGtz6sMj+x4CcMxr+vyOmkp5eMjXxF46jRCaGB0ZIdXXnxOAfdEG8WcNmON
Q8bEKcIr7FNu08UH6OFLom/blyLlAozct79aZJHDCdtbbkvr0TYbTakF4PsTrAyE
scSnUibr+i31/5S5gbhQMJKTx3Djl35dtPvzRzAPPK8qY2yKRUP1cDJInuXPgNpI
SYP2aHSoFUFcp8USfrNpetbt6nh1YY4Xu8qaPG5pdgODIT7Sqb+8Lid3Sl4KIqmh
pLSIookZnSzcpsDP0HrLS/qL0RGZwWnU1Sd48z7MiGxp1JoVB8wEivawnODuplde
uDhj+ICq/JKX/LrbxV2DqY3XPvStjGLOV20cpHJnVSlWWNHGo9+4LwpKZDV364pD
9IAbUx9mrIT0Zf3n1WOybcYjfikyX7JRJb98V+7PH63YuOcJI1TUVkFot9bXigvb
pOr56NTsVOF57ctJ1hSjP9ZDWR7XYESBwsCz2zNbD5h34I28EE5egtp6gjj9XtzW
CnZbSD3B/iO6Vp3SReBqTR7PxSvfSxnk6U7ZReMh8XxNiFT/tECzAyKee5CC2eut
CzrwQAWrAd3EjB4YO18FsYXpukhTs1rUOlvA6axrdOfjZ+LqQ4/+uVPzWUNxa/cQ
EeXflVuyp5LO0Ba4QbZNEDkMtNPmb7ie8l9ya6+DX53SiFiax2ezI4q7LP9nQldT
+6PFe+iu6Ctm+s10T1cP6ryjf3jV884CBSiZERO6oiuIXU2ncIXxfSJ/nvkZilSv
mSnKG8taO6OlkRQhayrktdEmArFR3/y2nlPlG5NNk0ZfE5uY+d7eDf5t/IypsF4f
wSVZN5D2L5gOl/gmyYQ6sazWi6T8mNfOT4evj32mr2yR1x35P+p6o4PrF5tunzfo
8RKz9Z8697dIiQC22j2u6R9F4l9nrs9n2uW9kiNiBG9rGDFIyjiUVcYiU0vyuKh/
vMuSVLq8neyOIkPUd9Dp0JX2AD3L0RDJ3GU4Jkcyi6MU3T8tdgiifHMhoSV+57CG
/biVNqnUCrkQHjcA1b0FKakKkd3iwXlunfguYs74vRGPgceNYrMstczIvB+Kilcm
vUua4ppPSOC2zHBF+D8kI8+z4KaGUh7K9VBqoXysTS6DLhcTmFkSWecD1vxMxv6B
vMsW2ScsRyd6hdFr6zE6YH2gdn12tgyzqUbxjMJVLp2X9JSszzRnczZTwM+oPByR
lzcqBiSafWrjrmL3nA04kA3MA0CSBd2BntjkS5/g5n3gU/kIj8fTSGvM6nrptwWZ
HaGaGoGS2y6ObY+8JGa/6SL43Uw/pGEJ8QbLCyZXs9Wjc5NhSO2SaOtr2eKK0uwb
KJ8WpyqAjBPu9M6DgIZVnO79T+vvtd10cJCR4rF67DF4rWcH72B5Ot7gNzj5Fni9
wMTkSgJ87sIEd8BXxz5YPbVAytNKL/5LCrxH1M7OvPoABwVY3qE0GZcyF5ZwxGCe
MbntuXDyclLbCvnpJsTiKNWUIDp9BsQcvcEY+XGIIqr1EdsHVzYxNY3uG9pvjbaH
XlAU/RvjP+4lekYlU0M6r0y9nHdPVacfjK1/LGNVbYT+
-----END ENCRYPTED PRIVATE KEY-----`;

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

const EC_KEY = string `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg+NirrP0T/b2Km/Cr
TzVesu8A8aDKtC15VhKwVGe9bZWhRANCAAQhDSXMvCo8EsOqn+gMle1HTuv0KrmU
6SIEX3FcLnmuCeqKPwfm1XQFsNRLSlYx49COGHwWo1A16cHZytm9Wp7y
-----END PRIVATE KEY-----`;

const RSA_KEY_PATH = "/tmp/bal_jwt_rsa_key.pem";
const RSA_ENCRYPTED_KEY_PATH = "/tmp/bal_jwt_rsa_enc_key.pem";
const RSA_CERT_PATH = "/tmp/bal_jwt_rsa_cert.pem";
const EC_KEY_PATH = "/tmp/bal_jwt_ec_key.pem";

function lastDot(string value) returns int {
    int index = -1;
    int? next = value.indexOf(".");
    while next is int {
        index = next;
        next = value.indexOf(".", next + 1);
    }
    return index;
}

function toBase64Url(byte[] value) returns string|error {
    byte[] urlSafe = [];
    foreach byte b in value.toBase64().toBytes() {
        if b == 43 {
            urlSafe.push(45);
        } else if b == 47 {
            urlSafe.push(95);
        } else if b != 61 {
            urlSafe.push(b);
        }
    }
    return string:fromBytes(urlSafe);
}

function headerSegment(string token) returns string {
    int? dot = token.indexOf(".");
    return dot is int ? token.substring(0, dot) : token;
}

function hmacMatches(string token, string secret) returns boolean|error {
    int last = lastDot(token);
    byte[] expected = check crypto:hmacSha256(token.substring(0, last).toBytes(), secret.toBytes());
    return check toBase64Url(expected) == token.substring(last + 1);
}

// RSA PKCS#1 v1.5 signatures are deterministic, so re-signing the assertion reproduces the token's signature.
function rsaMatches(string token, crypto:PrivateKey privateKey) returns boolean|error {
    int last = lastDot(token);
    byte[] expected = check crypto:signRsaSha256(token.substring(0, last).toBytes(), privateKey);
    return check toBase64Url(expected) == token.substring(last + 1);
}

function showError(string|jwt:Error result) {
    if result is jwt:Error {
        io:println(result.message(), " (cause: ", result.cause() is error, ")");
    } else {
        io:println("issued");
    }
}

public function main() returns error? {
    check io:fileWriteString(RSA_KEY_PATH, RSA_KEY);
    check io:fileWriteString(RSA_ENCRYPTED_KEY_PATH, RSA_ENCRYPTED_KEY);
    check io:fileWriteString(RSA_CERT_PATH, RSA_CERT);
    check io:fileWriteString(EC_KEY_PATH, EC_KEY);
    crypto:PrivateKey privateKey = check crypto:decodeRsaPrivateKeyFromKeyFile(RSA_KEY_PATH);

    jwt:IssuerConfig issuerConfig = {
        issuer: "wso2",
        username: "alice",
        audience: ["ballerina", "nutcracker"],
        jwtId: "id-1",
        keyId: "k1",
        customClaims: {
            "scp": "read write",
            "text": "a/b\"c\n\u{e9}",
            "nested": {"list": [1, 2.5, true, ()], "decimal": 1.50d},
            "small": 1.0e-7,
            "large": 12345678901234567.0
        },
        signatureConfig: {algorithm: jwt:HS256, config: "a-shared-secret"}
    };
    string token = check jwt:issue(issuerConfig);
    // The header serializes as jBallerina's toJsonString does: {"alg":"HS256", "typ":"JWT", "kid":"k1"}
    io:println(headerSegment(token)); // @output eyJhbGciOiJIUzI1NiIsICJ0eXAiOiJKV1QiLCAia2lkIjoiazEifQ
    io:println(check hmacMatches(token, "a-shared-secret")); // @output true
    [jwt:Header, jwt:Payload] decoded = check jwt:decode(token);
    jwt:Payload payload = decoded[1];
    io:println(payload.iss, " ", payload.sub, " ", payload.aud, " ", payload.jti); // @output wso2 alice ["ballerina","nutcracker"] id-1
    io:println(payload.exp - payload.iat, " ", payload.iat == payload.nbf); // @output 300 true
    io:println(payload["scp"], " | ", payload["text"] == "a/b\"c\n\u{e9}"); // @output read write | true
    io:println(payload["nested"]); // @output {"list":[1,2.5,true,null],"decimal":1.50}
    io:println(payload["small"], " ", payload["large"]); // @output 1.0E-7 12345678901234568

    string hs384 = check jwt:issue({signatureConfig: {algorithm: jwt:HS384, config: "secret"}});
    io:println(headerSegment(hs384)); // @output eyJhbGciOiJIUzM4NCIsICJ0eXAiOiJKV1QifQ
    string hs512 = check jwt:issue({signatureConfig: {algorithm: jwt:HS512, config: "secret"}});
    io:println(headerSegment(hs512)); // @output eyJhbGciOiJIUzUxMiIsICJ0eXAiOiJKV1QifQ

    string rs256 = check jwt:issue({issuer: "wso2", signatureConfig: {config: {keyFile: RSA_KEY_PATH}}});
    io:println(headerSegment(rs256), " ", check rsaMatches(rs256, privateKey)); // @output eyJhbGciOiJSUzI1NiIsICJ0eXAiOiJKV1QifQ true
    string rs384 = check jwt:issue({signatureConfig: {algorithm: jwt:RS384, config: {keyFile: RSA_ENCRYPTED_KEY_PATH, keyPassword: "keypass"}}});
    io:println(headerSegment(rs384)); // @output eyJhbGciOiJSUzM4NCIsICJ0eXAiOiJKV1QifQ
    string rs512 = check jwt:issue({signatureConfig: {algorithm: jwt:RS512, config: privateKey}});
    io:println(headerSegment(rs512)); // @output eyJhbGciOiJSUzUxMiIsICJ0eXAiOiJKV1QifQ
    string fromPrivateKey = check jwt:issue({signatureConfig: {config: privateKey}});
    io:println(check rsaMatches(fromPrivateKey, privateKey)); // @output true

    string unsigned = check jwt:issue({issuer: "wso2", expTime: 10.6});
    io:println(headerSegment(unsigned), " ", lastDot(unsigned) == unsigned.indexOf(".")); // @output eyJhbGciOiJub25lIiwgInR5cCI6IkpXVCJ9 true
    string none = check jwt:issue({signatureConfig: {algorithm: jwt:NONE, config: "ignored"}});
    io:println(headerSegment(none)); // @output eyJhbGciOiJub25lIiwgInR5cCI6IkpXVCJ9

    showError(jwt:issue({signatureConfig: {algorithm: jwt:RS256}})); // @output Signing JWT requires keystore information or private key information. (cause: false)
    showError(jwt:issue({signatureConfig: {config: "secret"}})); // @output Unsupported signing algorithm 'RS256'. (cause: false)
    showError(jwt:issue({signatureConfig: {algorithm: jwt:HS256, config: {keyFile: RSA_KEY_PATH}}})); // @output Unsupported signing algorithm 'HS256'. (cause: false)
    showError(jwt:issue({signatureConfig: {config: {keyFile: "/tmp/bal_jwt_missing.pem"}}})); // @output Failed to decode private key. (cause: true)
    showError(jwt:issue({signatureConfig: {config: {keyFile: EC_KEY_PATH}}})); // @output Failed to decode private key. (cause: true)
    crypto:PrivateKey ecKey = check crypto:decodeEcPrivateKeyFromKeyFile(EC_KEY_PATH);
    showError(jwt:issue({signatureConfig: {config: ecKey}})); // @output RSA private key signing failed for SHA256 algorithm. (cause: true)
    showError(jwt:issue({signatureConfig: {algorithm: jwt:RS384, config: ecKey}})); // @output RSA private key signing failed for SHA384 algorithm. (cause: true)
    showError(jwt:issue({signatureConfig: {algorithm: jwt:RS512, config: ecKey}})); // @output RSA private key signing failed for SHA512 algorithm. (cause: true)

    jwt:ClientSelfSignedJwtAuthProvider provider = new (issuerConfig);
    // Changing the caller's record after init does not change the provider's configuration.
    issuerConfig.issuer = "changed";
    string generated = check provider.generateToken();
    io:println((check jwt:decode(generated))[1].iss); // @output wso2
    jwt:ClientSelfSignedJwtAuthProvider failing = new ({signatureConfig: {config: {keyFile: "/tmp/bal_jwt_missing.pem"}}});
    showError(failing.generateToken()); // @output Failed to generate a self-signed JWT. (cause: true)
}
