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

import ballerina/crypto;
import ballerina/io;

public function main() returns error? {
    string certPem = string `
-----BEGIN CERTIFICATE-----
MIIC/zCCAeegAwIBAgIUBXoOP/FQ9aBoHTSUMU7hq5YHkTowDQYJKoZIhvcNAQEL
BQAwDzENMAsGA1UEAwwEdGVzdDAeFw0yNjA2MjQwOTQxNDFaFw0zNjA2MjEwOTQx
NDFaMA8xDTALBgNVBAMMBHRlc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEK
AoIBAQC1PPPztnW/FG3c7bpGpp7SFhjSfMFUef4mBtEWSc1ZRUkfhNb/fHIsoQMP
7PRv7kGE/ZxmKQ9KopDxoEvZ+yIndHsAmFoIz1d4qHFo9s/ZbGJjlPHI0rPbmkde
3/gSevQgs0p8zqS4HtFWdZR5yIYkY0ldI7bD7bTaYyMkxeBhg06nTbZ3UqqwKwv+
c38zDW+e1MqNdlN+XLWRHe9xVz05WoWypZWFCdfk4YeVoo7ccAfqNv6AiKWsk8VP
icoMz53kHoitlGRM6EGjLaVuiWn4V18N0HNYGM7zc743VxRd5d7VID2CO7mN2fiD
HGj65+bwQLkh5188V0abjiwbzhNlAgMBAAGjUzBRMB0GA1UdDgQWBBTccyUbTLdq
ZINgfGuVW/Olk4SfsjAfBgNVHSMEGDAWgBTccyUbTLdqZINgfGuVW/Olk4SfsjAP
BgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCfjarBf3GgaL3TpKg3
tBirLaYtl2wTuTyKgf7ZYYSpWxxyP0cs255s8Im/GcoLfb86WikeZ0lnbWvb7C4F
SobLlJ+ajRYr7TxVsmz6BNr+Dd04LSF9Mf2toZ4jy08/AdmbbmF7a+CZPIlsNSXI
uVasyeTYd0v52SJ9i+7cs1j+R5WBTxrptjv39KfXJe4/hbdIf8wQMtsZLAokFl7G
VVljwBVzGfmmzowf60HpSnfpEwAle7utOAmARLyf+hp3FNPRwqylDMcXEFD+/OJJ
M3vuL+Gkbuifdee3xayNRQOcT7wegwhqX18AgeJRP8JnksSCOqswYqQXI10chsgv
4uEG
-----END CERTIFICATE-----
`;
    string keyPem = string `
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC1PPPztnW/FG3c
7bpGpp7SFhjSfMFUef4mBtEWSc1ZRUkfhNb/fHIsoQMP7PRv7kGE/ZxmKQ9KopDx
oEvZ+yIndHsAmFoIz1d4qHFo9s/ZbGJjlPHI0rPbmkde3/gSevQgs0p8zqS4HtFW
dZR5yIYkY0ldI7bD7bTaYyMkxeBhg06nTbZ3UqqwKwv+c38zDW+e1MqNdlN+XLWR
He9xVz05WoWypZWFCdfk4YeVoo7ccAfqNv6AiKWsk8VPicoMz53kHoitlGRM6EGj
LaVuiWn4V18N0HNYGM7zc743VxRd5d7VID2CO7mN2fiDHGj65+bwQLkh5188V0ab
jiwbzhNlAgMBAAECggEAPWz/MaaxmaWO5sYb0D5AreuXVh+6VqtSHAlLbEZvNsZo
1inrxIOlHsMjio5A+n7B7hUWoPlhmWYnwf0WODcJiF3OIpGAUmQTvW05ot2j7Ijz
f9THbc0b8F4Fun4mUf0iKMMbh9lxsoWfZbJMNEpmTIbqIossMpOqLCpViu0V83Y+
jics/bavFPAV2XpEW/uj+iN1OUXmZgHtlxoekl5qeL2DEjfrqLYOzF+gfeJGf0cK
RQSpehhuxZo7RR8Twp3hr7sKWnEkzVg6LignEw1vsDU0KQ7lgkVjWlFInWHorohH
U2BgJgCbzl28qYC5yBg+eRmne1MKH+EmfVxBLvIvgQKBgQDxFlCryrjryTb598Ee
PMsuqd4lVhJ5AafwHvAKIhgml1uAwZT0hxOaZ9h8LaglbMy2I756RA1zGHIMqBYv
yCRRA5YNuobx9Zc2F817mcobdHoSg3RDaozIbIf7xRJwvd4vW8z+a4ucP9/ynURb
1VxEJvzN3FvU1mO5c8ZCTW29/QKBgQDAcukW+RYSTbj7jo5CpcQfDU+4YtPWSwFZ
mnme6EqDBsYKVq8Cz7+pebv0D8EyJJFGqgnkGFDd3gU4gA3Y2qneH5dY0UQhoqJT
tiFzmiWS3T6Re1eMKbjFY7E5LB59WyTMzFtdoFZhdDkN+WvEkuuWyM2GQKzmtI/y
CsBqZEIziQKBgQC6aknYfDk+wGiNImCmM9Xb8CdAcWx5OqmThyiOfUx1UqXDSmwW
I/gpdVC0vEz/G0CzObJIMiTAMU/Gr5XwPm6uYfp+BRPhNchFYGRXxVO8pPTbKeAV
XOcc9qazK/AVUwrhTbeVpqzeFZnhrG82HyVn4UmrGE+9pESaGoZbsClCNQKBgQCF
3ymf7nPZFbHpY9g4KoHMLAFZvX2o4xI0V43k6afzj4Gx7Wze4s9rwB/r/g2hqOha
JKyuu+989xXgoMuBH1LtDkLE6QWg9DZBTz/j38XlbPw6TXewK9G5lcjRgYxQHVfz
EvE3pvKP5j5OJ0Q9QQqbIGI/0ruz3MUJVUtWdxnKKQKBgD+XDq4K2S3Lu4TTDKlq
n2tibFB+z1Bf+UNeqXSH6l5hBetLFO1sxBCgEXJqdqZQzGcbwKiyxvR0Xq27avk6
XqWeJHCOAoR8Xuwe+xA8tfvr/cSGsrOK3xKUYnppx6wx2Goxsegyc8Uq9pxU6ub+
sMbVENrDBjGHSNDcwP4bsjzU
-----END PRIVATE KEY-----
`;
    string encKeyPem = string `
-----BEGIN ENCRYPTED PRIVATE KEY-----
MIIFLTBXBgkqhkiG9w0BBQ0wSjApBgkqhkiG9w0BBQwwHAQIba1oPEjXWZkCAggA
MAwGCCqGSIb3DQIJBQAwHQYJYIZIAWUDBAEqBBCpntLqaIEuOa6HfQoVEmtTBIIE
0Jz8FuHIArJ93w0mcqt3wgSvRGyTfOmCIiR3r5xSZLJwyGgMsJf3e8kD2A8mBdls
FPjRaaTCEUqGjZx6N2zFJSqzbzKnWqOn3LCYK1MfYsnFEfxTv+LClsv777gmUusa
BLmEoL46uQ2iw13Ow9a2AglS1wlG85gAUeVoAPK60UGk824hCQEzMJyOQZ/C8w8d
Iu6hR/aSbbyCvW9y2FNYFJ9EOqtROCeufx9Jkd8SZnvZnQDoPwDcjApwytGNRZbh
tjgpZ2VTRP/hLJuUT/FD8rurPL5UacWD1I5f0OMb8VPHmjP08RUTJJPb+Ey4zc3O
GaBUsf/TaW5ooVRBuqFWp9+/bHOiZFWUC3Arqzg5l8OF1vwD/56gaXy2mb42mbyG
JsmYNtPABk029IfFspiScFSggaDfMkN57NcQdVSvgT32sdRUuNNnX+4Z82PylJqu
5fjXiK+0wytipWIJNNMZygXrC5Z3BpLfVhp7YANNWhOuEPeBUyOhl3ueqGDom9+z
xILmHaetzvAR652kSoTDGdbVReFRfbVJlZRhneB60iNgObUlBQqqQVtBAa60Msjh
DqRn6ZNlDFfuvPlDrSS+lc6c8/dd+f507NBh22nNANNx02VOqNRb5/o2tPg0cXt5
MNsqtxw/NhYIJ0Ypc4K9da8nUndvgPfjiQZk3ghOwvN7apkwHDCEMcytqeJ30WNK
bTxP9ZJy0pEW6YsAzekuVozV5anm7R+MfvvoA9MEyp9KKwKfzu9/sf0nbMPfBHOQ
41L3ldzQJed40T/SJIqs18QeAM5Tz9VImGeq7RqWMayqFkVQl9zFWIWVVv09aTB0
CB3450jAktXOlE6bgu8wQML3PkNxT+3oLYwUWjqejIpM0/3uegGFSKYAgVNYsSCt
nRpAN+ISiIY8fFZ/zZYJuWr6tzKiFkyQg/8WWj2LELNsFNxmKDw6vXYXCT/VW9FD
ZseKHBvfxlvJ1tOeWiM6PtDonqVVjzbWq8WYs02C0CAovNe35BSfsTHxmzVwmwSr
lBphLE9pGDKARmGtXec5Lq6UX2nWti4+Y34gQ60mSR3A+db2PbITFxeqrFPb02h8
A51H4ZzqVORWSRp1GCCCLmUl7m1YIkKHDD3P6UI7QywiB2I9ALo/Ow3Hzd8gU2Gs
wiNT35vGUOkABaya7EQGkLvbFxb/M3t7U5yC+CCMz4ummFt4oLG/FuYFWE2MG2c1
Y4unteJrWPQVvLsWVPflw1LTSS0Ae6YUvBbK6b9KhPVw+i7xvD1xe31gVoBOK6ht
MaZ4HYhrGstGd4K8LwFmspFZ25CeZYdETxt/b3mL60WgQiayPka6lOA9N4pSgBLZ
YMcbgppoER9dZwaXI0ZK/50hVMozuOgWkcVe9eldKJ/eVVa8AKyd0wLPMeODKSUF
hmUofn1BvlFbG9+gK8sJJZcfbQ4qHPtkU2zDjQ+xDFOpqtImLg1ot224pPaFW+IZ
IUWtAnhDqLqtXHTQj6PbxzmbhiygtL4ID0qimS/IFb+Y9pT/IN8rdiiGYUicQbwu
Eubczj1asLiR3WmG1u2v/ntgx3FESVPFlpEMX66jA4DTWUF8aSNSNZIhKoIorIVZ
7zHOeuPbInRBgp4S7e5u4P6yRVVcCQiiETIIa78EvdjF
-----END ENCRYPTED PRIVATE KEY-----
`;

    byte[] data = [1, 2, 3, 4];

    // decodeRsaPublicKeyFromContent: PEM cert bytes obtained via io:fileReadBytes.
    check io:fileWriteString("/tmp/bal_crypto_content.pem", certPem);
    byte[] certBytes = check io:fileReadBytes("/tmp/bal_crypto_content.pem");
    crypto:PublicKey pub = check crypto:decodeRsaPublicKeyFromContent(certBytes);
    io:println((check crypto:encryptRsaEcb(data, pub)).length() > 0); // @output true

    // decodeRsaPrivateKeyFromContent: unencrypted PKCS#8 key bytes.
    check io:fileWriteString("/tmp/bal_crypto_content.pem", keyPem);
    byte[] keyBytes = check io:fileReadBytes("/tmp/bal_crypto_content.pem");
    crypto:PrivateKey pk = check crypto:decodeRsaPrivateKeyFromContent(keyBytes);
    io:println((check crypto:signRsaSha256(data, pk)).length() > 0); // @output true

    // decodeRsaPrivateKeyFromContent: encrypted key bytes + password.
    check io:fileWriteString("/tmp/bal_crypto_content.pem", encKeyPem);
    byte[] encBytes = check io:fileReadBytes("/tmp/bal_crypto_content.pem");
    crypto:PrivateKey encPk = check crypto:decodeRsaPrivateKeyFromContent(encBytes, "secret");
    io:println((check crypto:signRsaSha256(data, encPk)).length() > 0); // @output true

    // Malformed content is rejected.
    byte[] junk = [1, 2, 3];
    crypto:PublicKey|crypto:Error bad = crypto:decodeRsaPublicKeyFromContent(junk);
    io:println(bad is crypto:Error); // @output true

    // buildRsaPublicKey from hex modulus/exponent, and the invalid-hex error path.
    crypto:PublicKey built = check crypto:buildRsaPublicKey("B53CF3F3B675BF146DDCEDBA46A69ED21618D27CC15479FE2606D11649CD5945491F84D6FF7C722CA1030FECF46FEE4184FD9C66290F4AA290F1A04BD9FB2227747B00985A08CF5778A87168F6CFD96C626394F1C8D2B3DB9A475EDFF8127AF420B34A7CCEA4B81ED156759479C8862463495D23B6C3EDB4DA632324C5E061834EA74DB67752AAB02B0BFE737F330D6F9ED4CA8D76537E5CB5911DEF71573D395A85B2A5958509D7E4E18795A28EDC7007EA36FE8088A5AC93C54F89CA0CCF9DE41E88AD94644CE841A32DA56E8969F8575F0DD0735818CEF373BE3757145DE5DED5203D823BB98DD9F8831C68FAE7E6F040B921E75F3C57469B8E2C1BCE1365", "10001");
    io:println((check crypto:encryptRsaEcb(data, built)).length() > 0); // @output true
    crypto:PublicKey|crypto:Error badBuilt = crypto:buildRsaPublicKey("nothex", "10001");
    io:println(badBuilt is crypto:Error); // @output true
}
