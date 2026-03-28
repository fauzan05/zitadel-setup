# Lark OAuth2 adapter (for Zitadel + Lark)

Small HTTP service that sits between **Zitadel** and **Lark** so Zitadel can use Lark as an external IdP when only OAuth2 + userinfo APIs are available.

## Endpoints

| Method | Path        | Purpose |
|--------|-------------|---------|
| `POST` | `/token`    | Forwards to Lark token API (`authen/v2/oauth/token`). Request body may be JSON or `application/x-www-form-urlencoded`. Supports `client_id` / `client_secret` in body or `Authorization: Basic`. Response is pass-through from Lark (status + JSON). |
| `GET`  | `/userinfo` | Calls Lark user info API (`authen/v1/user_info`) with the incoming `Authorization` header and returns **normalized** OIDC-like JSON (`sub`, `given_name`, `family_name`, `email`, `name`, etc.). |

## Environment

| Variable           | Default | Description |
|--------------------|---------|-------------|
| `PORT`             | `4000`  | Listen address port (`:PORT`). |
| `LARK_TOKEN_URL`   | Lark v2 token URL | Override for tests or region-specific endpoints. |
| `LARK_USERINFO_URL`| Lark v1 user_info URL | Override if needed. |

## Zitadel: use **Generic OIDC**, not Generic OAuth

Zitadel’s **Generic OAuth2** provider uses an internal `UserMapper` whose `GetFirstName` / `GetLastName` / `GetEmail` etc. are **always empty** (only `GetID` reads your configured id attribute). Auto-create will then call `AddHumanUser` with an empty `GivenName`, which fails validation.

The **Generic OIDC** provider uses the standard OIDC `UserInfo` mapping (`given_name` → given name, `family_name` → family name, etc.), which matches what this proxy returns.

In the Zitadel Console:

1. Remove or stop using the **OAuth** IdP entry for Lark.
2. Add an identity provider of type **OIDC** (or “Generic OIDC” / similar wording in your console version).
3. Set:
   - **Authorization endpoint**: Lark’s authorize URL (unchanged).
   - **Token endpoint**: `http://lark-proxy:4000/token` (from Docker network).
   - **Userinfo endpoint**: `http://lark-proxy:4000/userinfo`.
   - Configure scopes, client id/secret, and callbacks as you already do for Lark.

If the console asks for an issuer / discovery URL, use the option to enter **manual** authorization / token / userinfo URLs if available.

## Docker

Built by `docker compose` in the parent repo (`build: ./lark-proxy`). No published host port by default; other containers on the `zitadel` network use the service name `lark-proxy`.
