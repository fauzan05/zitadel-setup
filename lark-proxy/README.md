# Lark OAuth2 adapter (for Zitadel + Lark)

Small HTTP service that sits between **Zitadel** and **Lark** so Zitadel can use Lark as an external IdP when only OAuth2 + userinfo APIs are available.

## Endpoints

| Method | Path        | Purpose |
|--------|-------------|---------|
| `GET`  | `/.well-known/openid-configuration` | OIDC discovery for ZITADEL Generic OIDC. Publishes issuer + authorization/token/userinfo/jwks endpoints. |
| `GET`  | `/oauth/v2/keys` | JWKS endpoint (empty key set; this flow relies on userinfo claims). |
| `POST` | `/token`    | Forwards to Lark token API (`authen/v2/oauth/token`). Request body may be JSON or `application/x-www-form-urlencoded`. Supports `client_id` / `client_secret` in body or `Authorization: Basic`. Response is pass-through from Lark (status + JSON). |
| `GET`  | `/userinfo` | Calls Lark user info API (`authen/v1/user_info`) with the incoming `Authorization` header and returns **normalized** OIDC-like JSON (`sub`, `preferred_username`, `given_name`, `family_name`, `email`, `email_verified`, `name`, etc.). |

## Environment

| Variable           | Default | Description |
|--------------------|---------|-------------|
| `PORT`             | `4000`  | Listen address port (`:PORT`). |
| `OIDC_ISSUER`      | `http://lark-proxy:4000` | Issuer URL exposed by discovery. Put this exact URL into the ZITADEL Generic OIDC **Issuer** field (or your public/internal equivalent). |
| `LARK_AUTHORIZE_URL` | Lark v1 authorize URL | Authorization endpoint shown in discovery (usually `https://accounts.larksuite.com/open-apis/authen/v1/authorize`). |
| `LARK_TOKEN_URL`   | Lark v2 token URL | Override for tests or region-specific endpoints. |
| `LARK_USERINFO_URL`| Lark v1 user_info URL | Override if needed. |

## Zitadel: use **Generic OIDC**, not Generic OAuth

Zitadel’s **Generic OAuth2** provider uses an internal `UserMapper` whose `GetFirstName` / `GetLastName` / `GetEmail` etc. are **always empty** (only `GetID` reads your configured id attribute). Auto-create will then call `AddHumanUser` with an empty `GivenName`, which fails validation.

The **Generic OIDC** provider uses the standard OIDC `UserInfo` mapping (`given_name` → given name, `family_name` → family name, etc.), which matches what this proxy returns.

In the Zitadel Console:

1. Remove or stop using the **OAuth** IdP entry for Lark.
2. Add an identity provider of type **OIDC** (or “Generic OIDC” / similar wording in your console version).
3. Set:
   - **Issuer**: `http://lark-proxy:4000` (or your `OIDC_ISSUER` value).
   - **Client ID / Secret**: your Lark app credentials.
   - **Scopes**: `openid profile email`.
   - Configure callbacks as you already do for Lark.

The proxy now exposes discovery at `/.well-known/openid-configuration`, so manual endpoint override is not required.

## Docker

Built by `docker compose` in the parent repo (`build: ./lark-proxy`). No published host port by default; other containers on the `zitadel` network use the service name `lark-proxy`.

## Userinfo contract (important)

The proxy enforces and normalizes these claims for Zitadel compatibility:

- `sub` (required): mapped from Lark `open_id`.
- `preferred_username`: email when available, fallback to `open_id`.
- `given_name` and `family_name`: guaranteed non-empty fallback from profile name or username.
- `email`: lower-cased when present.
- `email_verified`: true when email exists, false otherwise.

If upstream `open_id` is missing, `/userinfo` returns `502` because account linking is impossible without a stable subject.
