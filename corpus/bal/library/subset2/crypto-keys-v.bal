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
    string rsaKey = string `
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
    string rsaCert = string `
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
    string ecKey = string `
-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgaepZkIHyb5gE9B5c
7+Ng3vCUTYVAO0wM+19t6KURyX6hRANCAAQkLRdWZxRoNh8FdvtFp/fgQayHFznV
zoMlmOTdgPnFgmNlqSkDck6pJrbkjDPqLZeUekC5AHEWa56WPLoyKJDp
-----END PRIVATE KEY-----
`;
    string ecCert = string `
-----BEGIN CERTIFICATE-----
MIIBdjCCAR2gAwIBAgIUDIQSDugk7EpqSUG49OGo74ahhZswCgYIKoZIzj0EAwIw
ETEPMA0GA1UEAwwGZWN0ZXN0MB4XDTI2MDYyNDA5NDIyMloXDTM2MDYyMTA5NDIy
MlowETEPMA0GA1UEAwwGZWN0ZXN0MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
JC0XVmcUaDYfBXb7Raf34EGshxc51c6DJZjk3YD5xYJjZakpA3JOqSa25Iwz6i2X
lHpAuQBxFmueljy6MiiQ6aNTMFEwHQYDVR0OBBYEFNYU3/bKr48kb15dft9wucOT
kTOOMB8GA1UdIwQYMBaAFNYU3/bKr48kb15dft9wucOTkTOOMA8GA1UdEwEB/wQF
MAMBAf8wCgYIKoZIzj0EAwIDRwAwRAIgQoUn/hh7TJcuqsild7DYw9+JkG3CCzKQ
j8SfkAvXrHcCIDBDbfZxrJtPWany9a7fCZBKyWG8nNy2HkzSxYeJ2GHh
-----END CERTIFICATE-----
`;

    check io:fileWriteString("/tmp/bal_rsa_key.pem", rsaKey);
    check io:fileWriteString("/tmp/bal_rsa_cert.pem", rsaCert);
    check io:fileWriteString("/tmp/bal_ec_key.pem", ecKey);
    check io:fileWriteString("/tmp/bal_ec_cert.pem", ecCert);

    // --- RSA key/cert decoding ---
    crypto:PrivateKey rsaPk = check crypto:decodeRsaPrivateKeyFromKeyFile("/tmp/bal_rsa_key.pem");
    crypto:PublicKey rsaPub = check crypto:decodeRsaPublicKeyFromCertFile("/tmp/bal_rsa_cert.pem");
    io:println(rsaPk.algorithm);  // @output RSA
    io:println(rsaPub.algorithm); // @output RSA

    byte[] data = [1, 2, 3, 4, 5, 6, 7, 8];

    // --- RSA sign/verify across hash algorithms ---
    byte[] s256 = check crypto:signRsaSha256(data, rsaPk);
    io:println(check crypto:verifyRsaSha256Signature(data, s256, rsaPub)); // @output true
    byte[] s384 = check crypto:signRsaSha384(data, rsaPk);
    io:println(check crypto:verifyRsaSha384Signature(data, s384, rsaPub)); // @output true
    byte[] s512 = check crypto:signRsaSha512(data, rsaPk);
    io:println(check crypto:verifyRsaSha512Signature(data, s512, rsaPub)); // @output true
    byte[] s1 = check crypto:signRsaSha1(data, rsaPk);
    io:println(check crypto:verifyRsaSha1Signature(data, s1, rsaPub));     // @output true
    byte[] sMd5 = check crypto:signRsaMd5(data, rsaPk);
    io:println(check crypto:verifyRsaMd5Signature(data, sMd5, rsaPub));    // @output true
    byte[] sPss = check crypto:signRsaSsaPss256(data, rsaPk);
    io:println(check crypto:verifyRsaSsaPss256Signature(data, sPss, rsaPub)); // @output true

    // A signature over different data must not verify.
    io:println(check crypto:verifyRsaSha256Signature([9, 9, 9], s256, rsaPub)); // @output false

    // --- RSA encrypt/decrypt round-trip (default PKCS1 padding) ---
    byte[] ct = check crypto:encryptRsaEcb(data, rsaPub);
    io:println(check crypto:decryptRsaEcb(ct, rsaPk) == data); // @output true

    // --- RSA OAEP padding variants (each MGF1 hash) ---
    byte[] oaepMd5 = check crypto:encryptRsaEcb(data, rsaPub, crypto:OAEPwithMD5andMGF1);
    io:println(check crypto:decryptRsaEcb(oaepMd5, rsaPk, crypto:OAEPwithMD5andMGF1) == data); // @output true
    byte[] oaep1 = check crypto:encryptRsaEcb(data, rsaPub, crypto:OAEPWithSHA1AndMGF1);
    io:println(check crypto:decryptRsaEcb(oaep1, rsaPk, crypto:OAEPWithSHA1AndMGF1) == data); // @output true
    byte[] oaep256 = check crypto:encryptRsaEcb(data, rsaPub, crypto:OAEPWithSHA256AndMGF1);
    io:println(check crypto:decryptRsaEcb(oaep256, rsaPk, crypto:OAEPWithSHA256AndMGF1) == data); // @output true
    byte[] oaep384 = check crypto:encryptRsaEcb(data, rsaPub, crypto:OAEPwithSHA384andMGF1);
    io:println(check crypto:decryptRsaEcb(oaep384, rsaPk, crypto:OAEPwithSHA384andMGF1) == data); // @output true
    byte[] oaep512 = check crypto:encryptRsaEcb(data, rsaPub, crypto:OAEPwithSHA512andMGF1);
    io:println(check crypto:decryptRsaEcb(oaep512, rsaPk, crypto:OAEPwithSHA512andMGF1) == data); // @output true

    // --- EC key/cert decoding + ECDSA sign/verify ---
    crypto:PrivateKey ecPk = check crypto:decodeEcPrivateKeyFromKeyFile("/tmp/bal_ec_key.pem");
    crypto:PublicKey ecPub = check crypto:decodeEcPublicKeyFromCertFile("/tmp/bal_ec_cert.pem");
    byte[] e256 = check crypto:signSha256withEcdsa(data, ecPk);
    io:println(check crypto:verifySha256withEcdsaSignature(data, e256, ecPub)); // @output true
    byte[] e384 = check crypto:signSha384withEcdsa(data, ecPk);
    io:println(check crypto:verifySha384withEcdsaSignature(data, e384, ecPub)); // @output true

    // --- RSA encrypt using a private key (the PrivateKey|PublicKey key parameter
    // permits either); decrypt with the same private key. ---
    byte[] ctPriv = check crypto:encryptRsaEcb(data, rsaPk);
    io:println(check crypto:decryptRsaEcb(ctPriv, rsaPk) == data); // @output true

    // Decrypting malformed ciphertext fails.
    byte[]|crypto:Error decErr = crypto:decryptRsaEcb(data, rsaPk);
    io:println(decErr is crypto:Error); // @output true

    // Decrypting with a public key (rather than the private key) is rejected.
    byte[]|crypto:Error decPubErr = crypto:decryptRsaEcb(ct, rsaPub);
    io:println(decPubErr is crypto:Error); // @output true

    // A PSS signature over different data does not verify.
    io:println(check crypto:verifyRsaSsaPss256Signature([9, 9, 9], sPss, rsaPub)); // @output false

    // --- Passing a key of the wrong algorithm is rejected on every sign/verify. ---
    byte[]|crypto:Error rsaSignEc = crypto:signRsaSha256(data, ecPk);
    io:println(rsaSignEc is crypto:Error);                                        // @output true
    boolean|crypto:Error rsaVerifyEc = crypto:verifyRsaSha256Signature(data, s256, ecPub);
    io:println(rsaVerifyEc is crypto:Error);                                      // @output true
    byte[]|crypto:Error pssSignEc = crypto:signRsaSsaPss256(data, ecPk);
    io:println(pssSignEc is crypto:Error);                                        // @output true
    boolean|crypto:Error pssVerifyEc = crypto:verifyRsaSsaPss256Signature(data, sPss, ecPub);
    io:println(pssVerifyEc is crypto:Error);                                      // @output true
    byte[]|crypto:Error ecSignRsa = crypto:signSha256withEcdsa(data, rsaPk);
    io:println(ecSignRsa is crypto:Error);                                        // @output true
    boolean|crypto:Error ecVerifyRsa = crypto:verifySha256withEcdsaSignature(data, e256, rsaPub);
    io:println(ecVerifyRsa is crypto:Error);                                      // @output true

    // --- Key/cert decoding error paths. ---
    // Missing files.
    crypto:PrivateKey|crypto:Error rsaKeyMissing = crypto:decodeRsaPrivateKeyFromKeyFile("/tmp/bal_no_such.pem");
    io:println(rsaKeyMissing is crypto:Error);                                    // @output true
    crypto:PrivateKey|crypto:Error ecKeyMissing = crypto:decodeEcPrivateKeyFromKeyFile("/tmp/bal_no_such.pem");
    io:println(ecKeyMissing is crypto:Error);                                     // @output true
    crypto:PublicKey|crypto:Error rsaCertMissing = crypto:decodeRsaPublicKeyFromCertFile("/tmp/bal_no_such.pem");
    io:println(rsaCertMissing is crypto:Error);                                   // @output true
    crypto:PublicKey|crypto:Error ecCertMissing = crypto:decodeEcPublicKeyFromCertFile("/tmp/bal_no_such.pem");
    io:println(ecCertMissing is crypto:Error);                                    // @output true

    // Decoding a key/cert of the wrong algorithm.
    crypto:PrivateKey|crypto:Error ecAsRsaKey = crypto:decodeRsaPrivateKeyFromKeyFile("/tmp/bal_ec_key.pem");
    io:println(ecAsRsaKey is crypto:Error);                                       // @output true
    crypto:PrivateKey|crypto:Error rsaAsEcKey = crypto:decodeEcPrivateKeyFromKeyFile("/tmp/bal_rsa_key.pem");
    io:println(rsaAsEcKey is crypto:Error);                                       // @output true
    crypto:PublicKey|crypto:Error ecAsRsaCert = crypto:decodeRsaPublicKeyFromCertFile("/tmp/bal_ec_cert.pem");
    io:println(ecAsRsaCert is crypto:Error);                                      // @output true
    crypto:PublicKey|crypto:Error rsaAsEcCert = crypto:decodeEcPublicKeyFromCertFile("/tmp/bal_rsa_cert.pem");
    io:println(rsaAsEcCert is crypto:Error);                                      // @output true

    // Decoding from unparsable content.
    byte[] junk = [1, 2, 3];
    crypto:PrivateKey|crypto:Error junkKey = crypto:decodeRsaPrivateKeyFromContent(junk);
    io:println(junkKey is crypto:Error);                                          // @output true
    byte[] ecKeyBytes = check io:fileReadBytes("/tmp/bal_ec_key.pem");
    crypto:PrivateKey|crypto:Error ecAsRsaContent = crypto:decodeRsaPrivateKeyFromContent(ecKeyBytes);
    io:println(ecAsRsaContent is crypto:Error);                                   // @output true
    byte[] ecCertBytes = check io:fileReadBytes("/tmp/bal_ec_cert.pem");
    crypto:PublicKey|crypto:Error ecAsRsaCertContent = crypto:decodeRsaPublicKeyFromContent(ecCertBytes);
    io:println(ecAsRsaCertContent is crypto:Error);                               // @output true
}
