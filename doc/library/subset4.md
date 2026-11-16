# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/oauth2` package, providing an OAuth2 client provider that obtains and refreshes access tokens.

## [oauth2](https://github.com/ballerina-platform/module-ballerina-oauth2/blob/master/docs/spec/spec.md)

Types: `oauth2:ClientOAuth2Provider` (class), `oauth2:GrantConfig`, `oauth2:ClientCredentialsGrantConfig`, `oauth2:PasswordGrantConfig`, `oauth2:RefreshTokenGrantConfig`, `oauth2:JwtBearerGrantConfig`, `oauth2:RefreshConfig`, `oauth2:ClientConfiguration`, `oauth2:ClientAuth`, `oauth2:SecureSocket`, `oauth2:CertKey`, `oauth2:HttpVersion`, `oauth2:CredentialBearer`, `oauth2:Error`. Constants: `oauth2:INFER_REFRESH_CONFIG`, `oauth2:DEFAULT_CONNECT_TIMEOUT`, `oauth2:DEFAULT_REQ_TIMEOUT`; module variables `oauth2:globalConnectTimeout` and `oauth2:globalReqTimeout` (fixed at 15 and 30 seconds).

| Method | Notes |
|---|---|
| `new oauth2:ClientOAuth2Provider(grantConfig)` | Copies the grant configuration and requests the first access token, panicking with an `oauth2:Error` if that fails |
| `ClientOAuth2Provider.generateToken()` | Returns the cached access token until it expires (`expires_in` from the response, or `defaultTokenExpTime`, less `clockSkew`), then requests a new one: the client credentials grant repeats its token request, the password grant uses its `refreshConfig` (or the token URL and settings with `INFER_REFRESH_CONFIG`) and the cached refresh token, the refresh token grant uses the cached refresh token or the configured one, and the JWT bearer grant refreshes with the cached refresh token when a client ID and secret are set. Concurrent refreshes are serialized. A failure returns `oauth2:Error` (`Failed to generate OAuth2 token.`) with the cause |

Token requests are form-encoded POSTs carrying the grant parameters, the trimmed `scopes` joined by spaces, the trimmed `optionalParams`, and `customPayload`. The client ID and secret go in a Basic `Authorization` header (`AUTH_HEADER_BEARER`, the default) or the body (`POST_BODY_BEARER`); the password grant URL-encodes the username and password. Any `2xx` response is accepted and must be JSON with a string `access_token`. `customHeaders` are added to the request, except the restricted `Connection`, `Content-Length`, `Expect`, `Host` and `Upgrade`, which return `oauth2:Error`. `connectTimeout` and `reqTimeout` (in seconds, defaulting to the global timeouts) must be positive. A URL without a scheme gets `https://` when `secureSocket` is set and `http://` otherwise. TLS trusts the certificate file in `secureSocket.cert` (or any server with `disable`) and presents an `oauth2:CertKey` client certificate.

Not covered in this subset: `oauth2:ListenerOAuth2Provider` (token introspection), configurable global timeouts, and a `crypto:TrustStore`, a `crypto:KeyStore` or an encrypted private key in `secureSocket`.
