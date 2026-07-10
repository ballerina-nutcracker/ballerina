# Ballerina Crypto Library

## Overview

This module provides cryptographic operations for Ballerina programs. The full jBallerina `crypto` module covers hashing, HMAC, password hashing, symmetric encryption (AES), asymmetric encryption and signing (RSA, ECDSA), key derivation (HKDF), and post-quantum primitives (ML-KEM, ML-DSA, HPKE, PGP). Ballerina Nutcracker supports the core cryptographic subset.

## Key Functionalities

- **Hashing**: MD5, SHA-1, SHA-256, SHA-384, SHA-512, Keccak-256 with optional salt prepend; CRC32B checksum.
- **HMAC**: HMAC-MD5, HMAC-SHA1, HMAC-SHA256, HMAC-SHA384, HMAC-SHA512.
- **Password hashing**: BCrypt, Argon2id, PBKDF2 with hash/verify pairs.
- **AES encryption**: AES-CBC, AES-ECB, AES-GCM (128/256-bit keys; GCM tag size configurable in bits, default 128).
- **RSA**: Encrypt/decrypt (PKCS1 and OAEP padding), sign/verify (MD5, SHA1, SHA256, SHA384, SHA512, PSS-SHA256).
- **ECDSA**: Sign/verify with SHA-256 and SHA-384 (DER-encoded signatures).
- **Key loading**: RSA and EC private keys from PEM files or raw bytes; RSA and EC public keys from X.509 certificates (PEM files or raw bytes); PKCS12 keystores/truststores.
- **HKDF**: HKDF-SHA256 key derivation with optional salt and info.
- **Utilities**: Constant-time comparison of hash values (`HashValue = byte[]|string`).

## Examples

```ballerina
import ballerina/crypto;
import ballerina/io;

public function main() returns error? {
    // "Hello Ballerina"
    byte[] data = [72, 101, 108, 108, 111, 32, 66, 97, 108, 108, 101, 114, 105, 110, 97];

    // Hash
    io:println(crypto:hashSha256(data).length()); // 32

    // HMAC
    byte[] key = [115, 101, 99, 114, 101, 116, 107, 101, 121, 48, 49, 50, 51, 52, 53, 54];
    io:println((check crypto:hmacSha256(data, key)).length()); // 32

    // AES-GCM round-trip
    byte[] aesKey = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15];
    byte[] iv = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11];
    byte[] enc = check crypto:encryptAesGcm(data, aesKey, iv);
    byte[] dec = check crypto:decryptAesGcm(enc, aesKey, iv);
    io:println(crypto:equalConstantTime(dec, data)); // true

    // HKDF
    byte[] ikm = [105, 110, 112, 117, 116];
    byte[] derived = check crypto:hkdfSha256(ikm, 32);
    io:println(derived.length()); // 32
}
```

## Ballerina Nutcracker Support Status

This library is currently being migrated to Go to support Ballerina Nutcracker. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Hash functions (MD5, SHA-1, SHA-256, SHA-384, SHA-512) | Supported | `hashMd5`, `hashSha1`, `hashSha256`, `hashSha384`, `hashSha512`. Optional `salt byte[]` prepended before hashing. |
| Keccak-256 hash | Supported | `hashKeccak256`. Uses legacy Keccak-256 (pre-standardisation), not SHA3-256, matching jBallerina. |
| CRC32B checksum | Supported | `crc32b`. Returns 8-character uppercase hex string. |
| HMAC functions (MD5, SHA-1, SHA-256, SHA-384, SHA-512) | Supported | `hmacMd5`, `hmacSha1`, `hmacSha256`, `hmacSha384`, `hmacSha512`. |
| BCrypt password hashing and verification | Supported | `hashBcrypt`, `verifyBcrypt`. `workFactor` parameter supported. |
| Argon2id password hashing and verification | Supported | `hashArgon2`, `verifyArgon2`. Configurable iterations, memory, parallelism. Format: `$argon2id$v=19$m=<mem>,t=<iter>,p=<par>$<b64salt>$<b64hash>`. |
| PBKDF2 password hashing and verification | Supported | `hashPbkdf2`, `verifyPbkdf2`. SHA1, SHA256, SHA512 algorithms; configurable iteration count. Format: `$pbkdf2-{SHA1\|SHA256\|SHA512}$i=<iter>$<b64salt>$<b64hash>`. |
| AES-CBC encryption and decryption | Supported | `encryptAesCbc`, `decryptAesCbc`. PKCS7 padding always applied. See Notable Behavioural Changes. |
| AES-ECB encryption and decryption | Supported | `encryptAesEcb`, `decryptAesEcb`. PKCS7 padding always applied. See Notable Behavioural Changes. |
| AES-GCM encryption and decryption | Supported | `encryptAesGcm`, `decryptAesGcm`. `tagSize` in bits (default 128); valid sizes: 32, 64, 96, 104, 112, 120, 128. |
| RSA encryption and decryption | Supported | `encryptRsaEcb`, `decryptRsaEcb`. PKCS1 and OAEP (MD5, SHA1, SHA256, SHA384, SHA512) padding supported. |
| RSA PKCS1v15 signing | Supported | `signRsaMd5`, `signRsaSha1`, `signRsaSha256`, `signRsaSha384`, `signRsaSha512`. |
| RSA-PSS signing | Supported | `signRsaSsaPss256`. SHA-256 digest; uses `PSSSaltLengthEqualsHash` to match jBallerina. |
| RSA PKCS1v15 signature verification | Supported | `verifyRsaMd5Signature`, `verifyRsaSha1Signature`, `verifyRsaSha256Signature`, `verifyRsaSha384Signature`, `verifyRsaSha512Signature`. |
| RSA-PSS signature verification | Supported | `verifyRsaSsaPss256Signature`. |
| ECDSA signing | Supported | `signSha256withEcdsa`, `signSha384withEcdsa`. DER-encoded ASN.1 signatures matching jBallerina format. |
| ECDSA signature verification | Supported | `verifySha256withEcdsaSignature`, `verifySha384withEcdsaSignature`. |
| RSA and EC private key loading from keystore | Supported | `decodeRsaPrivateKeyFromKeyStore`, `decodeEcPrivateKeyFromKeyStore`. PKCS12 format. |
| RSA and EC private key loading from PEM file | Supported | `decodeRsaPrivateKeyFromKeyFile`, `decodeEcPrivateKeyFromKeyFile`. PEM format (PKCS8, PKCS1, EC); encrypted keys (PKCS12 PBE, PBES2) supported. |
| RSA private key loading from PEM bytes | Supported | `decodeRsaPrivateKeyFromContent`. |
| RSA and EC public key loading from truststore | Supported | `decodeRsaPublicKeyFromTrustStore`, `decodeEcPublicKeyFromTrustStore`. PKCS12 format with friendly-name alias lookup. |
| RSA and EC public key loading from certificate file | Supported | `decodeRsaPublicKeyFromCertFile`, `decodeEcPublicKeyFromCertFile`. PEM or DER X.509 certificate. |
| RSA public key loading from PEM bytes | Supported | `decodeRsaPublicKeyFromContent`. |
| RSA public key construction from modulus and exponent | Supported | `buildRsaPublicKey`. Hex-encoded modulus and exponent. |
| HKDF-SHA256 key derivation | Supported | `hkdfSha256`. Optional `salt` and `info` parameters. |
| Constant-time hash comparison | Supported | `equalConstantTime`. `HashValue = byte[]\|string`. |
| Module-level error type | Partially Supported | `crypto:Error` declared as a plain `error` alias; `distinct` error subtypes not yet supported. |
| ML-KEM-768 post-quantum key encapsulation | Not Yet Supported | `encapsulate`, `decapsulate` not yet implemented. |
| ML-DSA-65 post-quantum digital signature | Not Yet Supported | `signMlDsa65`, `verifyMlDsa65Signature` not yet implemented. |
| Hybrid public-key encryption | Not Yet Supported | `hybridEncrypt`, `hybridDecrypt` not yet implemented. |
| PGP operations | Not Yet Supported | `pgpEncrypt`, `pgpDecrypt`, `pgpSign`, `pgpVerify` not yet implemented. |
| EC public key loading from PEM bytes | Not Yet Supported | `decodeEcPublicKeyFromContent` not yet implemented. |

### Notable Behavioural Changes

- **AES-CBC and AES-ECB always apply PKCS7 padding.** jBallerina selects PKCS5 or no padding based on the `padding` parameter value; the Go-native version always applies PKCS7 padding for CBC and ECB modes regardless of the parameter — Go's `cipher` package does not expose a separate no-padding mode. Programs relying on `NONE` padding will produce incorrect output.
