# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/jwt` package, providing JSON Web Token issuing, decoding and validation.

## [jwt](https://github.com/ballerina-platform/module-ballerina-jwt/blob/master/docs/spec/spec.md)

Types: `jwt:ClientSelfSignedJwtAuthProvider` (class), `jwt:ListenerJwtAuthProvider` (class), `jwt:IssuerConfig`, `jwt:ValidatorConfig`, `jwt:ValidatorSignatureConfig`, `jwt:IssuerSignatureConfig`, `jwt:Header`, `jwt:Payload`, `jwt:SigningAlgorithm`, `jwt:Error`. Constants: `jwt:RS256`, `jwt:RS384`, `jwt:RS512`, `jwt:HS256`, `jwt:HS384`, `jwt:HS512`, `jwt:NONE`.

| Method | Notes |
|---|---|
| `jwt:issue(issuerConfig)` | Builds the header (`alg`, `typ` `JWT`, and `kid` from `keyId`) and payload (`exp` from `expTime`, default 300 seconds, plus `iat`, `nbf`, `iss`, `sub`, `aud`, `jti` and `customClaims`), Base64 URL encodes each as JSON with members in insertion order, and signs with HMAC for a `string` secret or RSA for a `crypto:KeyStore`, a private key file or a `crypto:PrivateKey`. `NONE`, or no `signatureConfig`, returns the unsigned `header.payload`. A missing key, a key that doesn't match the algorithm, or a key that fails to load returns `jwt:Error` |
| `jwt:decode(jwt)` | Splits the token at `.` (two or three segments are required, counting empty ones), Base64 URL decodes the header and payload (padded or unpadded) and parses them as JSON objects. JSON integers become `int` and other numbers `decimal`; `exp`, `nbf` and `iat` are read as integers, falling back to `0`, and an `aud` array is converted to `string[]`. An `alg` other than `RS256`–`RS512` or `HS256`–`HS512`, or a malformed segment, returns `jwt:Error` |
| `jwt:validate(jwt, validatorConfig)` | Decodes the token, then checks `username` (`sub`), `issuer` (`iss`), `audience` (any match between the configured and token audiences), `jwtId` (`jti`), `keyId` (`kid`) and `customClaims` (compared by their string form) when configured, and `exp` and `nbf` against the current time with `clockSkew` (in seconds). With a `signatureConfig`, the signature segment is required and verified against `secret` (HMAC), `certFile` (a certificate file or `crypto:PublicKey`; a certificate's validity period is checked) or `trustStoreConfig` (RSA); an algorithm that doesn't match the configured key returns `jwt:Error`. Without a `signatureConfig`, no signature is checked. `jwksConfig` returns a not-yet-supported `jwt:Error` |
| `new jwt:ListenerJwtAuthProvider(validatorConfig)` | Copies the validator configuration and creates the validated-token cache from `cacheConfig` |
| `ListenerJwtAuthProvider.authenticate(credential)` | Requires three `.`-separated segments, then returns a cached payload if it hasn't expired, or validates the token, caching the payload on success. A validation failure returns `jwt:Error` (`JWT validation failed.`) with the validation error as its cause |
| `new jwt:ClientSelfSignedJwtAuthProvider(issuerConfig)` | Copies the issuer configuration, so later changes to the caller's record have no effect |
| `ClientSelfSignedJwtAuthProvider.generateToken()` | Issues a token from the configuration; a failure returns `jwt:Error` (`Failed to generate a self-signed JWT.`) with the issuing error as its cause |

Not covered in this subset: JWKS-based signature validation.
