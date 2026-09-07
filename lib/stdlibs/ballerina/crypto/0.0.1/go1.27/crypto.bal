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

import ballerina/time;

// Represents the error type of the `crypto` module.
public type Error error;

// AES key size in bytes: 16 (AES-128), 24 (AES-192), or 32 (AES-256).
public type AesKeySize 16|24|32;

// Represents the result of a hybrid encryption operation.
//
// + encapsulatedSecret - Encapsulated secret bytes
// + cipherText - Encrypted data bytes
public type HybridEncryptionResult record {|
    byte[] encapsulatedSecret;
    byte[] cipherText;
|};

// Represents the result of a key encapsulation operation.
//
// + encapsulatedSecret - Encapsulated secret bytes
// + sharedSecret - Shared secret bytes
public type EncapsulationResult record {|
    byte[] encapsulatedSecret;
    byte[] sharedSecret;
|};

// PGP compression algorithm tags.
public enum CompressionAlgorithmTags {
    UNCOMPRESSED = "0",
    ZIP = "1",
    ZLIB = "2",
    BZIP2 = "3"
}

// PGP symmetric key algorithm tags.
public enum SymmetricKeyAlgorithmTags {
    NULL = "0",
    IDEA = "1",
    TRIPLE_DES = "2",
    CAST5 = "3",
    BLOWFISH = "4",
    SAFER = "5",
    DES = "6",
    AES_128 = "7",
    AES_192 = "8",
    AES_256 = "9",
    TWOFISH = "10",
    CAMELLIA_128 = "11",
    CAMELLIA_192 = "12",
    CAMELLIA_256 = "13"
}

// Options for PGP encryption.
//
// + compressionAlgorithm - Compression algorithm to use
// + symmetricKeyAlgorithm - Symmetric key algorithm to use
// + armor - Whether to ASCII-armor the output
// + withIntegrityCheck - Whether to add an integrity check packet
// + markForYourEyesOnly - Whether to mark the message for-your-eyes-only
public type Options record {|
    CompressionAlgorithmTags compressionAlgorithm = ZIP;
    SymmetricKeyAlgorithmTags symmetricKeyAlgorithm = AES_256;
    boolean armor = true;
    boolean withIntegrityCheck = true;
    boolean markForYourEyesOnly = false;
|};

// Key algorithm RSA.
public const RSA = "RSA";

// Key algorithm ML-KEM-768 (post-quantum).
public const MLKEM768 = "ML-KEM-768";

// Key algorithm ML-DSA-65 (post-quantum).
public const MLDSA65 = "ML-DSA-65";

// Represents the key algorithm.
public type KeyAlgorithm RSA|MLKEM768|MLDSA65;

// Represents a KeyStore.
//
// + path - Path to the KeyStore file
// + password - KeyStore password
public type KeyStore record {|
    string path;
    string password;
|};

// Represents a TrustStore.
//
// + path - Path to the TrustStore file
// + password - TrustStore password
public type TrustStore record {|
    string path;
    string password;
|};

// Represents a private key.
//
// + algorithm - Key algorithm
public type PrivateKey record {|
    KeyAlgorithm algorithm;
    never...;
|};

// Represents a public key.
//
// + algorithm - Key algorithm
// + certificate - Public key certificate
public type PublicKey record {|
    KeyAlgorithm algorithm;
    Certificate? certificate = ();
    never...;
|};

// Represents a public key certificate.
//
// + version - Certificate version
// + serial - Certificate serial number
// + issuer - Certificate issuer name
// + subject - Certificate subject name
// + notBefore - Certificate validity start time
// + notAfter - Certificate validity end time
// + signature - Certificate signature bytes
// + signingAlgorithm - Certificate signing algorithm OID
public type Certificate record {|
    int version;
    int serial;
    string issuer;
    string subject;
    time:Utc notBefore;
    time:Utc notAfter;
    byte[] signature;
    string signingAlgorithm;
|};

# Decodes an RSA private key from a PKCS12 KeyStore.
#
# + keyStore - KeyStore record with path and password
# + keyAlias - Alias of the key entry
# + keyPassword - Password of the key entry
# + return - PrivateKey or an Error
public isolated function decodeRsaPrivateKeyFromKeyStore(KeyStore keyStore, string keyAlias, string keyPassword)
        returns PrivateKey|Error = external;

# Decodes an EC private key from a PKCS12 KeyStore.
#
# + keyStore - KeyStore record with path and password
# + keyAlias - Alias of the key entry
# + keyPassword - Password of the key entry
# + return - PrivateKey or an Error
public isolated function decodeEcPrivateKeyFromKeyStore(KeyStore keyStore, string keyAlias, string keyPassword)
        returns PrivateKey|Error = external;

# Decodes an RSA private key from a PEM key file.
#
# + keyFile - Path to the key file
# + keyPassword - Optional password for encrypted keys
# + return - PrivateKey or an Error
public isolated function decodeRsaPrivateKeyFromKeyFile(string keyFile, string? keyPassword = ())
        returns PrivateKey|Error = external;

# Decodes an RSA private key from PEM-encoded content.
#
# + content - PEM-encoded key bytes
# + keyPassword - Optional password for encrypted keys
# + return - PrivateKey or an Error
public isolated function decodeRsaPrivateKeyFromContent(byte[] content, string? keyPassword = ())
        returns PrivateKey|Error = external;

# Decodes an EC private key from a PEM key file.
#
# + keyFile - Path to the key file
# + keyPassword - Optional password for encrypted keys
# + return - PrivateKey or an Error
public isolated function decodeEcPrivateKeyFromKeyFile(string keyFile, string? keyPassword = ())
        returns PrivateKey|Error = external;

# Decodes an RSA public key from a PKCS12 TrustStore.
#
# + trustStore - TrustStore record with path and password
# + keyAlias - Alias of the key entry
# + return - PublicKey or an Error
public isolated function decodeRsaPublicKeyFromTrustStore(TrustStore trustStore, string keyAlias)
        returns PublicKey|Error = external;

# Decodes an EC public key from a PKCS12 TrustStore.
#
# + trustStore - TrustStore record with path and password
# + keyAlias - Alias of the key entry
# + return - PublicKey or an Error
public isolated function decodeEcPublicKeyFromTrustStore(TrustStore trustStore, string keyAlias)
        returns PublicKey|Error = external;

# Decodes an RSA public key from a PEM certificate file.
#
# + certFile - Path to the certificate file
# + return - PublicKey or an Error
public isolated function decodeRsaPublicKeyFromCertFile(string certFile) returns PublicKey|Error = external;

# Decodes an RSA public key from PEM-encoded content.
#
# + content - PEM-encoded certificate bytes
# + return - PublicKey or an Error
public isolated function decodeRsaPublicKeyFromContent(byte[] content) returns PublicKey|Error = external;

# Decodes an EC public key from a PEM certificate file.
#
# + certFile - Path to the certificate file
# + return - PublicKey or an Error
public isolated function decodeEcPublicKeyFromCertFile(string certFile) returns PublicKey|Error = external;

# Builds an RSA public key from a modulus and exponent encoded as hexadecimal strings.
#
# + modulus - Hex-encoded modulus
# + exponent - Hex-encoded public exponent
# + return - PublicKey or an Error
public isolated function buildRsaPublicKey(string modulus, string exponent) returns PublicKey|Error = external;

// Algorithms supported by HMAC and PBKDF2 operations.
public enum HmacAlgorithm {
    SHA1,
    SHA256,
    SHA512
}

# Returns the MD5 hash of the given data, optionally with a salt prepended.
#
# + input - Value to be hashed
# + salt - Optional salt prepended before hashing
# + return - MD5 hash bytes
public isolated function hashMd5(byte[] input, byte[]? salt = ()) returns byte[] = external;

# Returns the SHA-1 hash of the given data, optionally with a salt prepended.
#
# + input - Value to be hashed
# + salt - Optional salt prepended before hashing
# + return - SHA-1 hash bytes
public isolated function hashSha1(byte[] input, byte[]? salt = ()) returns byte[] = external;

# Returns the SHA-256 hash of the given data, optionally with a salt prepended.
#
# + input - Value to be hashed
# + salt - Optional salt prepended before hashing
# + return - SHA-256 hash bytes
public isolated function hashSha256(byte[] input, byte[]? salt = ()) returns byte[] = external;

# Returns the SHA-384 hash of the given data, optionally with a salt prepended.
#
# + input - Value to be hashed
# + salt - Optional salt prepended before hashing
# + return - SHA-384 hash bytes
public isolated function hashSha384(byte[] input, byte[]? salt = ()) returns byte[] = external;

# Returns the SHA-512 hash of the given data, optionally with a salt prepended.
#
# + input - Value to be hashed
# + salt - Optional salt prepended before hashing
# + return - SHA-512 hash bytes
public isolated function hashSha512(byte[] input, byte[]? salt = ()) returns byte[] = external;

# Returns the Keccak-256 hash of the given data, optionally with a salt prepended.
#
# + input - Value to be hashed
# + salt - Optional salt prepended before hashing
# + return - Keccak-256 hash bytes
public isolated function hashKeccak256(byte[] input, byte[]? salt = ()) returns byte[] = external;

# Returns the CRC32B checksum of the given data as a hexadecimal string.
#
# + input - Value to be checksummed
# + return - CRC32B checksum hex string
public isolated function crc32b(byte[] input) returns string = external;

# Hashes a password using BCrypt.
#
# + password - Password string to hash
# + workFactor - BCrypt work factor (cost); defaults to 12
# + return - BCrypt hash string or an Error
public isolated function hashBcrypt(string password, int workFactor = 12) returns string|Error = external;

# Verifies a password against a BCrypt hash.
#
# + password - Password string to verify
# + hashedPassword - BCrypt hash string to compare against
# + return - true if the password matches, or an Error
public isolated function verifyBcrypt(string password, string hashedPassword) returns boolean|Error = external;

# Hashes a password using Argon2id.
#
# + password - Password string to hash
# + iterations - Number of iterations; defaults to 3
# + memory - Memory in KiB; defaults to 65536
# + parallelism - Degree of parallelism; defaults to 4
# + return - Argon2id hash string or an Error
public isolated function hashArgon2(string password, int iterations = 3, int memory = 65536, int parallelism = 4)
        returns string|Error = external;

# Verifies a password against an Argon2id hash.
#
# + password - Password string to verify
# + hashedPassword - Argon2id hash string to compare against
# + return - true if the password matches, or an Error
public isolated function verifyArgon2(string password, string hashedPassword) returns boolean|Error = external;

# Hashes a password using PBKDF2.
#
# + password - Password string to hash
# + iterations - Number of iterations; defaults to 10000
# + algorithm - HMAC algorithm to use; defaults to SHA256
# + return - PBKDF2 hash string or an Error
public isolated function hashPbkdf2(string password, int iterations = 10000, HmacAlgorithm algorithm = SHA256)
        returns string|Error = external;

# Verifies a password against a PBKDF2 hash.
#
# + password - Password string to verify
# + hashedPassword - PBKDF2 hash string to compare against
# + return - true if the password matches, or an Error
public isolated function verifyPbkdf2(string password, string hashedPassword) returns boolean|Error = external;

# Returns the HMAC using the MD5 hash function of the given data.
#
# + input - Value to be HMAC-ed
# + key - Key used for HMAC generation
# + return - HMAC bytes or an Error
public isolated function hmacMd5(byte[] input, byte[] key) returns byte[]|Error = external;

# Returns the HMAC using the SHA-1 hash function of the given data.
#
# + input - Value to be HMAC-ed
# + key - Key used for HMAC generation
# + return - HMAC bytes or an Error
public isolated function hmacSha1(byte[] input, byte[] key) returns byte[]|Error = external;

# Returns the HMAC using the SHA-256 hash function of the given data.
#
# + input - Value to be HMAC-ed
# + key - Key used for HMAC generation
# + return - HMAC bytes or an Error
public isolated function hmacSha256(byte[] input, byte[] key) returns byte[]|Error = external;

# Returns the HMAC using the SHA-384 hash function of the given data.
#
# + input - Value to be HMAC-ed
# + key - Key used for HMAC generation
# + return - HMAC bytes or an Error
public isolated function hmacSha384(byte[] input, byte[] key) returns byte[]|Error = external;

# Returns the HMAC using the SHA-512 hash function of the given data.
#
# + input - Value to be HMAC-ed
# + key - Key used for HMAC generation
# + return - HMAC bytes or an Error
public isolated function hmacSha512(byte[] input, byte[] key) returns byte[]|Error = external;

# Derives a key using HKDF with SHA-256.
#
# + input - Input key material
# + length - Length of the derived key in bytes
# + salt - Optional salt bytes; defaults to empty
# + info - Optional context and application-specific info bytes; defaults to empty
# + return - Derived key bytes or an Error
public isolated function hkdfSha256(byte[] input, int length, byte[] salt = [], byte[] info = [])
        returns byte[]|Error = external;

# Signs data using RSA with MD5 digest.
#
# + input - Data to sign
# + privateKey - RSA private key
# + return - Signature bytes or an Error
public isolated function signRsaMd5(byte[] input, PrivateKey privateKey) returns byte[]|Error = external;

# Signs data using RSA with SHA-1 digest.
#
# + input - Data to sign
# + privateKey - RSA private key
# + return - Signature bytes or an Error
public isolated function signRsaSha1(byte[] input, PrivateKey privateKey) returns byte[]|Error = external;

# Signs data using RSA with SHA-256 digest.
#
# + input - Data to sign
# + privateKey - RSA private key
# + return - Signature bytes or an Error
public isolated function signRsaSha256(byte[] input, PrivateKey privateKey) returns byte[]|Error = external;

# Signs data using RSA with SHA-384 digest.
#
# + input - Data to sign
# + privateKey - RSA private key
# + return - Signature bytes or an Error
public isolated function signRsaSha384(byte[] input, PrivateKey privateKey) returns byte[]|Error = external;

# Signs data using RSA with SHA-512 digest.
#
# + input - Data to sign
# + privateKey - RSA private key
# + return - Signature bytes or an Error
public isolated function signRsaSha512(byte[] input, PrivateKey privateKey) returns byte[]|Error = external;

# Signs data using RSA-SSA-PSS with SHA-256 digest and MGF1.
#
# + input - Data to sign
# + privateKey - RSA private key
# + return - Signature bytes or an Error
public isolated function signRsaSsaPss256(byte[] input, PrivateKey privateKey) returns byte[]|Error = external;

# Signs data using ECDSA with SHA-256 digest.
#
# + input - Data to sign
# + privateKey - EC private key
# + return - DER-encoded signature bytes or an Error
public isolated function signSha256withEcdsa(byte[] input, PrivateKey privateKey) returns byte[]|Error = external;

# Signs data using ECDSA with SHA-384 digest.
#
# + input - Data to sign
# + privateKey - EC private key
# + return - DER-encoded signature bytes or an Error
public isolated function signSha384withEcdsa(byte[] input, PrivateKey privateKey) returns byte[]|Error = external;

# Verifies an RSA-MD5 signature.
#
# + data - Original data
# + signature - Signature bytes to verify
# + publicKey - RSA public key
# + return - true if the signature is valid, or an Error
public isolated function verifyRsaMd5Signature(byte[] data, byte[] signature, PublicKey publicKey)
        returns boolean|Error = external;

# Verifies an RSA-SHA-1 signature.
#
# + data - Original data
# + signature - Signature bytes to verify
# + publicKey - RSA public key
# + return - true if the signature is valid, or an Error
public isolated function verifyRsaSha1Signature(byte[] data, byte[] signature, PublicKey publicKey)
        returns boolean|Error = external;

# Verifies an RSA-SHA-256 signature.
#
# + data - Original data
# + signature - Signature bytes to verify
# + publicKey - RSA public key
# + return - true if the signature is valid, or an Error
public isolated function verifyRsaSha256Signature(byte[] data, byte[] signature, PublicKey publicKey)
        returns boolean|Error = external;

# Verifies an RSA-SHA-384 signature.
#
# + data - Original data
# + signature - Signature bytes to verify
# + publicKey - RSA public key
# + return - true if the signature is valid, or an Error
public isolated function verifyRsaSha384Signature(byte[] data, byte[] signature, PublicKey publicKey)
        returns boolean|Error = external;

# Verifies an RSA-SHA-512 signature.
#
# + data - Original data
# + signature - Signature bytes to verify
# + publicKey - RSA public key
# + return - true if the signature is valid, or an Error
public isolated function verifyRsaSha512Signature(byte[] data, byte[] signature, PublicKey publicKey)
        returns boolean|Error = external;

# Verifies an RSA-SSA-PSS-256 signature.
#
# + data - Original data
# + signature - Signature bytes to verify
# + publicKey - RSA public key
# + return - true if the signature is valid, or an Error
public isolated function verifyRsaSsaPss256Signature(byte[] data, byte[] signature, PublicKey publicKey)
        returns boolean|Error = external;

# Verifies an ECDSA signature with SHA-256 digest.
#
# + data - Original data
# + signature - DER-encoded signature bytes to verify
# + publicKey - EC public key
# + return - true if the signature is valid, or an Error
public isolated function verifySha256withEcdsaSignature(byte[] data, byte[] signature, PublicKey publicKey)
        returns boolean|Error = external;

# Verifies an ECDSA signature with SHA-384 digest.
#
# + data - Original data
# + signature - DER-encoded signature bytes to verify
# + publicKey - EC public key
# + return - true if the signature is valid, or an Error
public isolated function verifySha384withEcdsaSignature(byte[] data, byte[] signature, PublicKey publicKey)
        returns boolean|Error = external;

// No padding.
public const NONE = "NONE";

// PKCS5 padding (equivalent to PKCS7 in AES context).
public const PKCS5 = "PKCS5";

// RSA PKCS#1 v1.5 padding.
public const PKCS1 = "PKCS1";

// RSA OAEP with MD5 and MGF1 padding.
public const OAEPwithMD5andMGF1 = "OAEPwithMD5andMGF1";

// RSA OAEP with SHA-1 and MGF1 padding.
public const OAEPWithSHA1AndMGF1 = "OAEPWithSHA1AndMGF1";

// RSA OAEP with SHA-256 and MGF1 padding.
public const OAEPWithSHA256AndMGF1 = "OAEPWithSHA256AndMGF1";

// RSA OAEP with SHA-384 and MGF1 padding.
public const OAEPwithSHA384andMGF1 = "OAEPwithSHA384andMGF1";

// RSA OAEP with SHA-512 and MGF1 padding.
public const OAEPwithSHA512andMGF1 = "OAEPwithSHA512andMGF1";

// AES padding mode.
public type AesPadding NONE|PKCS5;

// RSA padding mode.
public type RsaPadding PKCS1|OAEPwithMD5andMGF1|OAEPWithSHA1AndMGF1|OAEPWithSHA256AndMGF1|OAEPwithSHA384andMGF1|OAEPwithSHA512andMGF1;

# Encrypts a byte array using RSA in ECB mode.
#
# + input - Data to encrypt
# + key - RSA public or private key
# + padding - RSA padding mode; defaults to PKCS1
# + return - Encrypted bytes or an Error
public isolated function encryptRsaEcb(byte[] input, PrivateKey|PublicKey key, RsaPadding padding = PKCS1)
        returns byte[]|Error = external;

# Encrypts a byte array using AES in CBC mode.
#
# + input - Data to encrypt
# + key - AES key bytes (16, 24, or 32 bytes)
# + iv - Initialization vector (16 bytes)
# + padding - AES padding mode; defaults to PKCS5
# + return - Encrypted bytes or an Error
public isolated function encryptAesCbc(byte[] input, byte[] key, byte[] iv, AesPadding padding = PKCS5)
        returns byte[]|Error = external;

# Encrypts a byte array using AES in ECB mode.
#
# + input - Data to encrypt
# + key - AES key bytes (16, 24, or 32 bytes)
# + padding - AES padding mode; defaults to PKCS5
# + return - Encrypted bytes or an Error
public isolated function encryptAesEcb(byte[] input, byte[] key, AesPadding padding = PKCS5)
        returns byte[]|Error = external;

# Encrypts a byte array using AES in GCM mode.
#
# + input - Data to encrypt
# + key - AES key bytes (16, 24, or 32 bytes)
# + iv - Initialization vector (12 bytes recommended)
# + padding - AES padding mode; defaults to NONE
# + tagSize - Authentication tag size in bits; defaults to 128
# + return - Encrypted bytes (ciphertext + tag) or an Error
public isolated function encryptAesGcm(byte[] input, byte[] key, byte[] iv, AesPadding padding = NONE,
        int tagSize = 128) returns byte[]|Error = external;

# Decrypts a byte array using RSA in ECB mode.
#
# + input - Data to decrypt
# + key - RSA public or private key
# + padding - RSA padding mode; defaults to PKCS1
# + return - Decrypted bytes or an Error
public isolated function decryptRsaEcb(byte[] input, PrivateKey|PublicKey key, RsaPadding padding = PKCS1)
        returns byte[]|Error = external;

# Decrypts a byte array using AES in CBC mode.
#
# + input - Data to decrypt
# + key - AES key bytes (16, 24, or 32 bytes)
# + iv - Initialization vector (16 bytes)
# + padding - AES padding mode; defaults to PKCS5
# + return - Decrypted bytes or an Error
public isolated function decryptAesCbc(byte[] input, byte[] key, byte[] iv, AesPadding padding = PKCS5)
        returns byte[]|Error = external;

# Decrypts a byte array using AES in ECB mode.
#
# + input - Data to decrypt
# + key - AES key bytes (16, 24, or 32 bytes)
# + padding - AES padding mode; defaults to PKCS5
# + return - Decrypted bytes or an Error
public isolated function decryptAesEcb(byte[] input, byte[] key, AesPadding padding = PKCS5)
        returns byte[]|Error = external;

# Decrypts a byte array using AES in GCM mode.
#
# + input - Data to decrypt (ciphertext + tag)
# + key - AES key bytes (16, 24, or 32 bytes)
# + iv - Initialization vector used during encryption
# + padding - AES padding mode; defaults to PKCS5 (ignored in GCM)
# + tagSize - Authentication tag size in bits; defaults to 128
# + return - Decrypted bytes or an Error
public isolated function decryptAesGcm(byte[] input, byte[] key, byte[] iv, AesPadding padding = PKCS5,
        int tagSize = 128) returns byte[]|Error = external;

// Represents a hash value as either a byte array or a string.
public type HashValue byte[]|string;

# Compares two hash values in constant time to prevent timing attacks.
#
# + value - Hash value to compare
# + expectedValue - Expected hash value
# + return - true if both values are equal
public isolated function equalConstantTime(HashValue value, HashValue expectedValue) returns boolean =
        external;
