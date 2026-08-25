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
    byte[] plaintext = [72, 101, 108, 108, 111, 87, 111, 114, 108, 100]; // "HelloWorld"
    byte[] key = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]; // 16-byte key
    byte[] iv = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16]; // 16-byte IV
    byte[] iv12 = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12];               // 12-byte GCM nonce

    // CBC round-trip (PKCS5 padding).
    byte[] cbc = check crypto:encryptAesCbc(plaintext, key, iv);
    io:println(cbc.length() > plaintext.length());                  // @output true
    io:println(check crypto:decryptAesCbc(cbc, key, iv) == plaintext); // @output true

    // ECB round-trip (PKCS5 padding).
    byte[] ecb = check crypto:encryptAesEcb(plaintext, key);
    io:println(check crypto:decryptAesEcb(ecb, key) == plaintext);  // @output true

    // GCM round-trip (no padding; default 128-bit tag).
    byte[] gcm = check crypto:encryptAesGcm(plaintext, key, iv12);
    io:println(check crypto:decryptAesGcm(gcm, key, iv12) == plaintext); // @output true

    // GCM with a non-default (96-bit) authentication tag length.
    byte[] gcm96 = check crypto:encryptAesGcm(plaintext, key, iv12, crypto:NONE, 96);
    io:println(check crypto:decryptAesGcm(gcm96, key, iv12, crypto:NONE, 96) == plaintext); // @output true

    // An invalid AES key length is rejected by every mode.
    byte[] badKey = [0, 1, 2, 3, 4]; // 5 bytes, not a valid AES key size
    byte[]|crypto:Error cbcErr = crypto:encryptAesCbc(plaintext, badKey, iv);
    io:println(cbcErr is crypto:Error);                                  // @output true
    byte[]|crypto:Error ecbErr = crypto:encryptAesEcb(plaintext, badKey);
    io:println(ecbErr is crypto:Error);                                  // @output true
    byte[]|crypto:Error gcmErr = crypto:encryptAesGcm(plaintext, badKey, iv12);
    io:println(gcmErr is crypto:Error);                                  // @output true

    // Decrypting a block with invalid PKCS7 padding fails (pkcs7Unpad).
    byte[] zeros = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0];
    byte[]|crypto:Error padErr = crypto:decryptAesCbc(zeros, key, iv);
    io:println(padErr is crypto:Error);                                  // @output true
    // Ciphertext whose length is not a multiple of the block size fails.
    byte[]|crypto:Error sizeErr = crypto:decryptAesCbc([1, 2, 3], key, iv);
    io:println(sizeErr is crypto:Error);                                 // @output true

    // An invalid AES key length is rejected on the decrypt side too.
    byte[]|crypto:Error cbcDecErr = crypto:decryptAesCbc(cbc, badKey, iv);
    io:println(cbcDecErr is crypto:Error);                               // @output true
    byte[]|crypto:Error ecbDecErr = crypto:decryptAesEcb(ecb, badKey);
    io:println(ecbDecErr is crypto:Error);                               // @output true

    // ECB decrypt rejects a non-block-multiple length and invalid padding.
    byte[]|crypto:Error ecbSizeErr = crypto:decryptAesEcb([1, 2, 3], key);
    io:println(ecbSizeErr is crypto:Error);                              // @output true
    byte[]|crypto:Error ecbPadErr = crypto:decryptAesEcb(zeros, key);
    io:println(ecbPadErr is crypto:Error);                               // @output true

    // A zero-length ciphertext fails PKCS7 unpadding.
    byte[]|crypto:Error emptyErr = crypto:decryptAesCbc([], key, iv);
    io:println(emptyErr is crypto:Error);                                // @output true

    // GCM rejects an invalid authentication tag size on both sides.
    byte[]|crypto:Error gcmEncTagErr = crypto:encryptAesGcm(plaintext, key, iv12, crypto:NONE, 48);
    io:println(gcmEncTagErr is crypto:Error);                            // @output true
    byte[]|crypto:Error gcmDecTagErr = crypto:decryptAesGcm(gcm, key, iv12, crypto:NONE, 48);
    io:println(gcmDecTagErr is crypto:Error);                            // @output true

    // GCM rejects a non-standard nonce+tag combination (16-byte IV with a
    // 96-bit tag matches neither the standard nonce nor the standard tag size).
    byte[]|crypto:Error gcmEncCombo = crypto:encryptAesGcm(plaintext, key, iv, crypto:NONE, 96);
    io:println(gcmEncCombo is crypto:Error);                             // @output true
    byte[]|crypto:Error gcmDecCombo = crypto:decryptAesGcm(gcm, key, iv, crypto:NONE, 96);
    io:println(gcmDecCombo is crypto:Error);                             // @output true

    // GCM decrypt rejects a bad key length and, with a valid key, a failed
    // authentication tag (wrong key over otherwise-valid ciphertext).
    byte[]|crypto:Error gcmDecKeyErr = crypto:decryptAesGcm(gcm, badKey, iv12);
    io:println(gcmDecKeyErr is crypto:Error);                            // @output true
    byte[] key2 = [15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0];
    byte[]|crypto:Error gcmAuthErr = crypto:decryptAesGcm(gcm, key2, iv12);
    io:println(gcmAuthErr is crypto:Error);                              // @output true
}
