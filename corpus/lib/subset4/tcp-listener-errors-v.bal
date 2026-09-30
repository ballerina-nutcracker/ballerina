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

import ballerina/file;
import ballerina/io;
import ballerina/tcp;

service class Svc {
    *tcp:Service;
}

const SERVER_CERT = string `
-----BEGIN CERTIFICATE-----
MIIDJTCCAg2gAwIBAgIUFrHCWR5issu0JSqz/JUQgGfcU10wDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTI2MDYxODA0NTg0MVoXDTM2MDYx
NTA0NTg0MVowFDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEArZ/3QGK1A/V3KqkzJbEgh9DoyBUd0uUcpb1wtXdp2iCf
uNEvIYcMLblYfywmrbojdEHEAhSDXR5NhBjR2/luWRm4+caEBwixrqUvHeLLNEic
TR6ihQYOGJ12AF7xJ7nFe/cvdwTz1nVTg6c/4erJwijHkoUIlezfCeiAVzuuUZxq
M3BD5kppk3c9kLe5SFrV4//hTaTEzI0bCqgKDbgTo/qUFlezSRNCgUDgVRv81PuW
tuBdnXi4KkxBsYgb2AOfMsUBM+7aNmpfKj9gZFiFc0Cbh4yrL3agdKV6SASzBqhi
15MXJOhm1qQhvYSydzlrWq6M7FbPCMO3pImBH0tDyQIDAQABo28wbTAdBgNVHQ4E
FgQU3YWyixJPnyQER1jumA21d4hVPCkwHwYDVR0jBBgwFoAU3YWyixJPnyQER1ju
mA21d4hVPCkwDwYDVR0TAQH/BAUwAwEB/zAaBgNVHREEEzARgglsb2NhbGhvc3SH
BH8AAAEwDQYJKoZIhvcNAQELBQADggEBAAvtKZ7kVNDA3YGenCFn+puugK7LrxYQ
n7Mzn9opaZJgzp4YCuYZRas5nfNj6gOnFEW1sCjdSO3fx/GtOCSuGp+/QSFW8Loa
G6StiZHgYdfp50U2AJU+/NqFy4gpgHMe+W/oOoFRZFuPxKgfxDZOvk75VrPecEud
A9NCwI67jtUUN9x2EjZF2A5USdxLqPcUIKMhLTXnrgpF1P83eGJXS3ZbZcdoyFMa
D8a2Xvv+ScO7sXEOh0y5DLYZk/5MUQacZIu1AN7gUxi/QPklcBT+Rw8d0L5djEwW
7d9CPY6Rvc+UI1qfVVfa3q5zeXWWMUaWr722cV7weO6aPAtbGP290yk=
-----END CERTIFICATE-----
`;

const OTHER_KEY = string `
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCvZd3Eqo10Hu2N
MedVINej0jfZTSiZHGvE2TGUMWkaAak7lo93FsIId5OkGxomYAAneSq6CywuqnNh
uGAe+n0/Uq65YWYRSOc3LOesXln3oAxKL2bAmpgPmoOQDEj4Rg5uqmD4XL+l9bAI
TUEhEY9Iv3+BmfFhvF89fI2jTn8WwzOWOIbqNQLDezSjZtBZ10IONczGafytKB9N
2EifehcgIUhBpNPah5qrm/1boH+tWQ26QqPHIEfK1EeUEmblPE5QR/G/ldNgajID
VNPyVxjKtMq+2Wobr2PhOsLtTmjIU8rD/S1CjM1BmYX723k+Xk76xCf7FStiCoEi
5p9VxEVLAgMBAAECggEAYvkTU+d9nHnbSCJRzEqVBB8aXqdYclWERnbcsCfVi9RG
H5bM68FOyiJXkHGuptrjG2MXBhZJ3ydVpl2496IXTMUmTPs18ij2N79CF5TCJaYN
aUsoKyTRC3UAKieUgWxetnmz8FG93zMtGvIk/zj+ZmmQ1Jv+hk0aXQLMlcs0ka2z
BQpCMAdv2+wcIttFioLGM93ha1roDC/96TEumd7z4I9ei0zVTRrdbuJAPYNxvuEQ
yhP3h77Vxzmva7Ze2mQ7YQo+YxWNvAPjntMo2gXBGlomdNRFkKRxspCdj8waXP6a
kNMY0I95SLOoT5dzuYXLe+9AEZ3vQprJgmz4nhTPkQKBgQDgUP/eFT4JHnVMSVMk
Gtmc7NyRfZoiz9aum6CrHC3ndll86Sr9KH6MS+2jibTvfKuqVunMFa8gLm+0+wrR
hiKYpXsQB4zApk6stOQhjO8tKwL5KGcEVxuCGxZwpQstwU3MpJb6KtQp26/d6Myn
/EmjPdx7o1Nskix/Lsjqv08vkwKBgQDILAkSpZcGBsTESTfBxsXfZjLrrmdlIVTG
sx+aR5uzD+KeQUFeLPTjI0HI66Agk61JZKg2H+T8H551+HihZ4ZE+zUcRhlOv3C+
7lnTxmgHwYltaadGjFGTfNg1O5N3xX7ZJ1d1C6Hyjiyai9Q5MTwZ6TQnFgqwNXC6
ijSuSbR2aQKBgQCKoaSwZSJ00k0d+4cVnj51XHGovR5ge2CrguUWOxEnpp4Bs7zs
vnUPUfGpC3c2Ny+E07gkeBZGZkyFZk02VpbVDhhzPKdBfMbMt/LxxT8gBCAPSmf4
Z7lJfGIu9eFlwrzJR8NDEusSf4I+PB0XiLUd3SKBitwJEoHLjGeBpQBUXwKBgE5T
TMmLGek1xNC5haPBT7P2nX4zJCoiPLXzaS5JTtdmI9qg1h8voDMwnJEuAd49atUJ
GfjzyfoNtUecDje59oQPd9y0YTq7FjbftS50uEgwpsG1Ucr9JP59H2zU/wMiuFvo
VQRrpNbwYuumQ0iNx2KsKcgRi+UfRBj581svY/nhAoGBAMCTlFjAfquV/q6+8rTG
l5XA0vV2Tjd0uSf311YBp+Ah68XEdpwPJcFDXrk8ebCx1/dDFuIGNjU4r3s5Blqs
21N/t2ttWGypmyL5OFsezSmb+a5o12hitk+79+EDiyEDsX6zZuCP1OT0gVtWF9kF
SbSx8Qr4X5czvd/6ZZgMnVmq
-----END PRIVATE KEY-----
`;

function startResult(string certFile, string keyFile) returns string {
    tcp:Listener|tcp:Error l = new (21902, secureSocket = {key: {certFile, keyFile}});
    if l is tcp:Error {
        return "init: " + l.message();
    }
    error? s = l.'start();
    return s is error ? s.message() : "started";
}

public function main() returns error? {
    tcp:Client|tcp:Error c = new ("127.0.0.1", 1, secureSocket = {cert: "/nonexistent/ca.pem"});
    io:println(c is tcp:Error ? c.message() : "connected"); // @output File does not contain valid certificates: /nonexistent/ca.pem
    string empty = check file:createTemp();
    tcp:Client|tcp:Error e = new ("127.0.0.1", 1, secureSocket = {cert: empty});
    io:println(e is tcp:Error && e.message() == "File does not contain valid certificates: " + empty); // @output true
    string garbage = check file:createTemp();
    check io:fileWriteString(garbage, "hello\n");
    tcp:Client|tcp:Error g1 = new ("127.0.0.1", 1, secureSocket = {cert: garbage});
    io:println(g1 is tcp:Error && g1.message() == "File does not contain valid certificates: " + garbage); // @output true

    string cert = check file:createTemp();
    check io:fileWriteString(cert, SERVER_CERT);
    io:println(startResult("/nonexistent/c.pem", "/nonexistent/k.pem")); // @output File does not contain valid certificates: /nonexistent/c.pem
    io:println(startResult(cert, "/nonexistent/k.pem")); // @output File does not contain valid private key: /nonexistent/k.pem
    io:println(startResult(garbage, cert) == "File does not contain valid certificates: " + garbage); // @output true
    io:println(startResult(cert, garbage) == "File does not contain valid private key: " + garbage); // @output true
    // jBallerina starts with a key that does not match the certificate (ballerina-platform/ballerina-library#9927).
    string otherKey = check file:createTemp();
    check io:fileWriteString(otherKey, OTHER_KEY);
    io:println(startResult(cert, otherKey) == "File does not contain a private key matching the certificate: " + otherKey); // @output true
    check file:remove(empty);
    check file:remove(garbage);
    check file:remove(cert);
    check file:remove(otherKey);

    tcp:Listener|tcp:Error l = new (21901);
    if l is tcp:Error {
        return l;
    }
    error? g = l.gracefulStop();
    io:println(g is error ? g.message() : "ok"); // @output Unable to initialize the tcp listener.
    error? r1 = l.attach(new Svc());
    error? r2 = l.attach(new Svc());
    io:println(r1 is (), " ", r2 is error); // @output true true
    // jBallerina detaches the attached service here (ballerina-library#9700).
    error? d = l.detach(new Svc());
    io:println(d is error); // @output true
}
