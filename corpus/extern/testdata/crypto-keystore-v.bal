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
    crypto:KeyStore ks = {path: "testdata/crypto-keystore.p12", password: "secret"};

    // Recover the RSA private key from the PKCS#12 keystore and use it to sign.
    crypto:PrivateKey pk = check crypto:decodeRsaPrivateKeyFromKeyStore(ks, "ballerina", "secret");
    byte[] sig = check crypto:signRsaSha256([1, 2, 3, 4], pk);
    io:println(sig.length() > 0); // @output true

    // Recover the public key from the same file used as a trust store.
    crypto:TrustStore ts = {path: "testdata/crypto-keystore.p12", password: "secret"};
    crypto:PublicKey pub = check crypto:decodeRsaPublicKeyFromTrustStore(ts, "ballerina");
    byte[] ct = check crypto:encryptRsaEcb([1, 2, 3, 4], pub);
    io:println(ct.length() > 0); // @output true

    // A wrong keystore password fails key recovery.
    crypto:KeyStore badKs = {path: "testdata/crypto-keystore.p12", password: "wrong"};
    crypto:PrivateKey|crypto:Error badKey = crypto:decodeRsaPrivateKeyFromKeyStore(badKs, "ballerina", "wrong");
    io:println(badKey is crypto:Error); // @output true

    // A missing keystore file fails.
    crypto:TrustStore missing = {path: "testdata/does-not-exist.p12", password: "secret"};
    crypto:PublicKey|crypto:Error missingErr = crypto:decodeRsaPublicKeyFromTrustStore(missing, "ballerina");
    io:println(missingErr is crypto:Error); // @output true

    // An EC key recovered via the RSA decoder fails: the store decodes fine,
    // but the recovered key is not RSA.
    crypto:KeyStore ecKs = {path: "testdata/crypto-keystore-ec.p12", password: "secret"};
    crypto:PrivateKey|crypto:Error notRsaKey = crypto:decodeRsaPrivateKeyFromKeyStore(ecKs, "ballerina-ec", "secret");
    io:println(notRsaKey is crypto:Error); // @output true

    // Recover the EC private key from the EC keystore and sign with it.
    crypto:PrivateKey ecPk = check crypto:decodeEcPrivateKeyFromKeyStore(ecKs, "ballerina-ec", "secret");
    byte[] ecSig = check crypto:signSha256withEcdsa([1, 2, 3, 4], ecPk);
    io:println(ecSig.length() > 0); // @output true

    // Recover the EC public key from the same file used as a trust store and
    // verify the signature just produced.
    crypto:TrustStore ecTs = {path: "testdata/crypto-keystore-ec.p12", password: "secret"};
    crypto:PublicKey ecPub = check crypto:decodeEcPublicKeyFromTrustStore(ecTs, "ballerina-ec");
    io:println(check crypto:verifySha256withEcdsaSignature([1, 2, 3, 4], ecSig, ecPub)); // @output true

    // The RSA keystore decoded through the EC decoder fails: the recovered key
    // is not EC.
    crypto:PrivateKey|crypto:Error notEcKey = crypto:decodeEcPrivateKeyFromKeyStore(ks, "ballerina", "secret");
    io:println(notEcKey is crypto:Error); // @output true

    // The EC trust store decoded through the RSA public-key decoder fails: the
    // certificate holds an EC key, not RSA.
    crypto:PublicKey|crypto:Error notRsaPub = crypto:decodeRsaPublicKeyFromTrustStore(ecTs, "ballerina-ec");
    io:println(notRsaPub is crypto:Error); // @output true

    // A missing keystore file fails on the private-key path too.
    crypto:KeyStore missingKs = {path: "testdata/does-not-exist.p12", password: "secret"};
    crypto:PrivateKey|crypto:Error missingKeyErr = crypto:decodeRsaPrivateKeyFromKeyStore(missingKs, "ballerina", "secret");
    io:println(missingKeyErr is crypto:Error); // @output true
}
