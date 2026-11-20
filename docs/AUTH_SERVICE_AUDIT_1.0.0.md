# UMMU Auth Service audit — v1.0.0

**Version source:** `APP_VERSION=1.0.0` in `.env.example`  
**Scope:** static review of JWT issuance/validation, user registration, Google
OIDC configuration, and Docker image construction.  
**Related SDK documentation:** [`ummu-go-auth v0.2.0`](../../ummu-go-outh/docs/v0.2.0.md)

## Findings

### High — Signing key and environment file are copied into the runtime image

[`Dockerfile`](../Dockerfile) copies both `.env` and `jwt_private.pem` into the
final image. The Compose service also mounts the key read-only and injects its
environment file, but those runtime mounts do not remove copies embedded in
the image layers. There is no `.dockerignore` in the service build context.
Anyone who can pull or inspect a built image may be able to recover the
signing key or configuration from its layers.

**Recommendation:** do not copy `.env` or private keys into any image layer.
Inject configuration and the private key at runtime through the deployment's
secret mechanism; exclude local secrets from the Docker build context and
rotate any key that has been distributed in an image.

### Medium — Registration accepts passwords without a minimum-strength check

[`UserRegisterRequest`](../internal/domain/dto/user_dto.go) validates that the
password is present but does not enforce a minimum length or strength policy.
The API documentation also demonstrates a one-character password. Hashing
protects stored passwords but does not prevent weak credentials from being
chosen.

**Recommendation:** introduce and document an explicit password policy,
enforce it consistently at registration and password-change boundaries, and
add login throttling at the service or trusted gateway boundary. Account for
the bcrypt 72-byte input limit documented by the API.

### Low — Google redirect URL is required even when Google login is disabled

[`setConfig`](../internal/config/config.go) rejects an empty
`GOOGLE_REDIRECT_URL` unconditionally, although Google OIDC is otherwise
enabled only when both client credentials are configured. The returned error
also names `GOOGLE_FRONTEND_CALLBACK_URL` instead of the field being checked.
Deployments that intentionally disable Google login must still configure an
unused redirect URL, and the error message can mislead operators.

**Recommendation:** require the provider redirect URL only when Google OIDC
credentials are configured, and make the validation error identify
`GOOGLE_REDIRECT_URL`.

## SDK compatibility

The current Auth Service token generator emits `iss`, `sub`, `aud`, `iat`,
`exp`, `jti`, and `principal_type`, satisfying the core `ummu-go-auth v0.2.0`
validation contract. User tokens carry RBAC access contexts and no OAuth
scope. Client tokens carry a matching UUID `sub`/`client_id` and requested
OAuth scopes. The SDK is not currently a dependency of Auth Service; this is a
review of token-contract compatibility, not evidence that the SDK is used by
the service itself.

## Verification and limitations

- `go vet ./...` completed successfully for Auth Service.
- `go test ./...` compiled packages, but Windows Application Control blocked
  execution of generated test binaries for some packages in this environment.
- No source-code remediation was made as part of this audit.
