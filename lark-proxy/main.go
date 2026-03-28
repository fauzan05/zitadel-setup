package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	defaultPort            = "4000"
	defaultLarkTokenURL    = "https://open.larksuite.com/open-apis/authen/v2/oauth/token"
	defaultLarkUserInfoURL = "https://open.larksuite.com/open-apis/authen/v1/user_info"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	tokenURL := os.Getenv("LARK_TOKEN_URL")
	if tokenURL == "" {
		tokenURL = defaultLarkTokenURL
	}
	userInfoURL := os.Getenv("LARK_USERINFO_URL")
	if userInfoURL == "" {
		userInfoURL = defaultLarkUserInfoURL
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/token", handleToken(tokenURL))
	mux.HandleFunc("/userinfo", handleUserInfo(userInfoURL))

	addr := ":" + port
	log.Printf("lark-proxy listening on %s (token=%s userinfo=%s)", addr, tokenURL, userInfoURL)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

// tokenBody holds fields we forward to Lark (JSON body).
type tokenBody struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
}

func handleToken(larkTokenURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fmt.Printf("[/token][ERROR] method not allowed: %s\n", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := readRequestBody(r)
		if err != nil {
			fmt.Printf("[/token][ERROR] invalid request body: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		clientID, clientSecret := body.ClientID, body.ClientSecret
		if clientID == "" && r.Header.Get("Authorization") != "" {
			if u, p, ok := parseBasicAuth(r.Header.Get("Authorization")); ok {
				clientID, clientSecret = u, p
			}
		}

		out := tokenBody{
			GrantType:    body.GrantType,
			Code:         body.Code,
			RefreshToken: body.RefreshToken,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURI:  body.RedirectURI,
		}
		log.Printf("[/token] grant_type=%q has_code=%t has_refresh_token=%t has_client_id=%t has_redirect_uri=%t",
			out.GrantType, out.Code != "", out.RefreshToken != "", out.ClientID != "", out.RedirectURI != "")

		payload, err := json.Marshal(out)
		if err != nil {
			fmt.Printf("[/token][ERROR] marshal failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, larkTokenURL, bytes.NewReader(payload))
		if err != nil {
			fmt.Printf("[/token][ERROR] create upstream request failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("[/token][ERROR] upstream request failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		upstreamBody, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("[/token][ERROR] read upstream body failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		log.Printf("[/token] upstream status=%d body=%s", resp.StatusCode, truncateForLog(string(upstreamBody), 800))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(upstreamBody)
	}
}

// readRequestBody parses JSON or x-www-form-urlencoded into tokenBody.
func readRequestBody(r *http.Request) (tokenBody, error) {
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return tokenBody{}, err
	}
	_ = r.Body.Close()

	var out tokenBody
	if strings.Contains(ct, "application/json") {
		if err := json.Unmarshal(raw, &out); err != nil {
			return tokenBody{}, err
		}
		return out, nil
	}

	if strings.Contains(ct, "application/x-www-form-urlencoded") || ct == "" {
		vals, err := url.ParseQuery(string(raw))
		if err != nil {
			return tokenBody{}, err
		}
		out.GrantType = vals.Get("grant_type")
		out.Code = vals.Get("code")
		out.RefreshToken = vals.Get("refresh_token")
		out.ClientID = vals.Get("client_id")
		out.ClientSecret = vals.Get("client_secret")
		out.RedirectURI = vals.Get("redirect_uri")
		if out.GrantType != "" || out.Code != "" || out.RefreshToken != "" || out.ClientID != "" {
			return out, nil
		}
	}

	if err := json.Unmarshal(raw, &out); err != nil {
		return tokenBody{}, err
	}
	return out, nil
}

func parseBasicAuth(h string) (user, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(h, prefix) {
		return "", "", false
	}
	b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(h, prefix))
	if err != nil {
		return "", "", false
	}
	cs := string(b)
	user, pass, found := strings.Cut(cs, ":")
	if !found {
		return "", "", false
	}
	return user, pass, true
}

// larkUserInfoRaw matches Lark API envelope + data (subset).
type larkUserInfoRaw struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type larkUserData struct {
	OpenID    string  `json:"open_id"`
	EnName    string  `json:"en_name"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url"`
}

// normalizedUserInfo is what Zitadel expects (OIDC-like claims).
type normalizedUserInfo struct {
	Sub               string  `json:"sub"`
	OpenID            string  `json:"open_id"`
	PreferredUsername string  `json:"preferred_username"`
	GivenName         string  `json:"given_name"`
	FamilyName        string  `json:"family_name"`
	Name              string  `json:"name"`
	Email             string  `json:"email"`
	Picture           *string `json:"picture,omitempty"`
}

func handleUserInfo(larkUserInfoURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			fmt.Printf("[/userinfo][ERROR] method not allowed: %s\n", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, larkUserInfoURL, nil)
		if err != nil {
			fmt.Printf("[/userinfo][ERROR] create upstream request failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if auth := r.Header.Get("Authorization"); auth != "" {
			req.Header.Set("Authorization", auth)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("[/userinfo][ERROR] upstream request failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("[/userinfo][ERROR] read upstream body failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("[/userinfo] upstream status=%d body=%s", resp.StatusCode, truncateForLog(string(body), 800))

		var raw larkUserInfoRaw
		if err := json.Unmarshal(body, &raw); err != nil {
			log.Printf("[/userinfo] upstream body not json envelope, pass-through")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write(body)
			return
		}

		if len(raw.Data) == 0 || string(raw.Data) == "null" {
			fmt.Printf("[/userinfo][ERROR] missing raw.data code=%d msg=%q\n", raw.Code, raw.Msg)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(body)
			return
		}

		var d larkUserData
		if err := json.Unmarshal(raw.Data, &d); err != nil {
			fmt.Printf("[/userinfo][ERROR] decode raw.data failed: %v\n", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(body)
			return
		}

		nameParts := strings.Fields(strings.TrimSpace(firstNonEmpty(d.EnName, d.Name)))
		firstName := ""
		lastName := ""
		if len(nameParts) > 0 {
			firstName = nameParts[0]
			lastName = strings.Join(nameParts[1:], " ")
			if lastName == "" {
				lastName = firstName
			}
		}
		username := d.OpenID
		if d.Email != "" {
			if i := strings.IndexByte(d.Email, '@'); i > 0 {
				username = d.Email[:i]
			}
		}
		if firstName == "" {
			// Zitadel auto-creation requires GivenName length 1..200.
			// Fallback to a stable non-empty value when Lark profile names are absent.
			firstName = username
		}
		if lastName == "" {
			lastName = firstName
		}
		displayName := firstNonEmpty(d.EnName, d.Name)
		if strings.TrimSpace(displayName) == "" {
			displayName = firstName
		}

		out := normalizedUserInfo{
			Sub:               d.OpenID,
			OpenID:            d.OpenID,
			PreferredUsername: username,
			GivenName:         firstName,
			FamilyName:        lastName,
			Name:              displayName,
			Email:             d.Email,
			Picture:           d.AvatarURL,
		}
		log.Printf("[/userinfo] normalized names preferred_username=%q given_name=%q family_name=%q name=%q", out.PreferredUsername, out.GivenName, out.FamilyName, out.Name)
		log.Printf("[/userinfo] normalized subject=%q has_email=%t has_picture=%t", out.Sub, out.Email != "", out.Picture != nil)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(out)
	}
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
