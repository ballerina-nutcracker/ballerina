# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/jwt` package, providing JSON Web Token issuing and decoding.

## [jwt](https://github.com/ballerina-platform/module-ballerina-jwt/blob/master/docs/spec/spec.md)

Types: `jwt:ClientSelfSignedJwtAuthProvider` (class), `jwt:IssuerConfig`, `jwt:IssuerSignatureConfig`, `jwt:Header`, `jwt:Payload`, `jwt:SigningAlgorithm`, `jwt:Error`. Constants: `jwt:RS256`, `jwt:RS384`, `jwt:RS512`, `jwt:HS256`, `jwt:HS384`, `jwt:HS512`, `jwt:NONE`.

| Method | Notes |
|---|---|
| `jwt:issue(issuerConfig)` | Builds the header (`alg`, `typ` `JWT`, and `kid` from `keyId`) and payload (`exp` from `expTime`, default 300 seconds, plus `iat`, `nbf`, `iss`, `sub`, `aud`, `jti` and `customClaims`), Base64 URL encodes each as JSON with members in insertion order, and signs with HMAC for a `string` secret or RSA for a `crypto:KeyStore`, a private key file or a `crypto:PrivateKey`. `NONE`, or no `signatureConfig`, returns the unsigned `header.payload`. A missing key, a key that doesn't match the algorithm, or a key that fails to load returns `jwt:Error` |
| `jwt:decode(jwt)` | Splits the token at `.` (two or three segments are required, counting empty ones), Base64 URL decodes the header and payload (padded or unpadded) and parses them as JSON objects. JSON integers become `int` and other numbers `decimal`; `exp`, `nbf` and `iat` are read as integers, falling back to `0`, and an `aud` array is converted to `string[]`. An `alg` other than `RS256`–`RS512` or `HS256`–`HS512`, or a malformed segment, returns `jwt:Error` |
| `new jwt:ClientSelfSignedJwtAuthProvider(issuerConfig)` | Copies the issuer configuration, so later changes to the caller's record have no effect |
| `ClientSelfSignedJwtAuthProvider.generateToken()` | Issues a token from the configuration; a failure returns `jwt:Error` (`Failed to generate a self-signed JWT.`) with the issuing error as its cause |

Not covered in this subset: `jwt:validate`, `jwt:ListenerJwtAuthProvider` and JWKS-based validation.
