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
    string password = "s3cr3t-passw0rd";

    // BCrypt: hash, verify correct, reject wrong.
    string bcryptHash = check crypto:hashBcrypt(password);
    io:println(bcryptHash.length() > 0);                              // @output true
    io:println(check crypto:verifyBcrypt(password, bcryptHash));      // @output true
    io:println(check crypto:verifyBcrypt("wrong", bcryptHash));       // @output false

    // Argon2: hash (custom params), verify correct, reject wrong.
    string argon2Hash = check crypto:hashArgon2(password);
    io:println(argon2Hash.length() > 0);                             // @output true
    io:println(check crypto:verifyArgon2(password, argon2Hash));     // @output true
    io:println(check crypto:verifyArgon2("wrong", argon2Hash));      // @output false

    // PBKDF2: default (SHA256) and SHA512 variants, verify both.
    string pbkdf2Hash = check crypto:hashPbkdf2(password);
    io:println(pbkdf2Hash.length() > 0);                            // @output true
    io:println(check crypto:verifyPbkdf2(password, pbkdf2Hash));    // @output true
    io:println(check crypto:verifyPbkdf2("wrong", pbkdf2Hash));     // @output false

    string pbkdf2Sha512 = check crypto:hashPbkdf2(password, 10000, crypto:SHA512);
    io:println(check crypto:verifyPbkdf2(password, pbkdf2Sha512));  // @output true

    string pbkdf2Sha1 = check crypto:hashPbkdf2(password, 10000, crypto:SHA1);
    io:println(check crypto:verifyPbkdf2(password, pbkdf2Sha1));    // @output true
}
