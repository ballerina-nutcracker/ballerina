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

public function main() {
    byte[] data = [1, 2, 3, 4];

    // A PrivateKey/PublicKey record can be constructed directly, without going
    // through a decode function, so it carries no associated native key. Every
    // operation on such a record must fail with a crypto:Error rather than
    // panic or silently succeed.
    crypto:PrivateKey pk = {algorithm: crypto:RSA};
    crypto:PublicKey pub = {algorithm: crypto:RSA};

    byte[]|crypto:Error rsaSig = crypto:signRsaSha256(data, pk);
    io:println(rsaSig is crypto:Error); // @output true

    boolean|crypto:Error rsaVerify = crypto:verifyRsaSha256Signature(data, data, pub);
    io:println(rsaVerify is crypto:Error); // @output true

    byte[]|crypto:Error rsaCipher = crypto:encryptRsaEcb(data, pub);
    io:println(rsaCipher is crypto:Error); // @output true

    byte[]|crypto:Error ecSig = crypto:signSha256withEcdsa(data, pk);
    io:println(ecSig is crypto:Error); // @output true
}
