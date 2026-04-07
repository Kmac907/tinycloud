package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tinycloud/internal/config"
)

type Service struct {
	cfg config.Config
}

type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	ExtExpires  int    `json:"ext_expires_in"`
	ExpiresOn   string `json:"expires_on"`
	NotBefore   string `json:"not_before"`
	Resource    string `json:"resource,omitempty"`
	Scope       string `json:"scope,omitempty"`
}

func NewService(cfg config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) IssueToken(audience, subject string, lifetime time.Duration) (Token, error) {
	if audience == "" {
		audience = s.cfg.TokenAudience
	}
	if subject == "" {
		subject = s.cfg.TokenSubject
	}
	if lifetime <= 0 {
		lifetime = time.Hour
	}

	now := time.Now().UTC()
	nbf := now.Unix()
	exp := now.Add(lifetime).Unix()

	claims := map[string]any{
		"aud": audience,
		"iss": s.cfg.EffectiveTokenIssuer(),
		"sub": subject,
		"tid": s.cfg.TenantID,
		"oid": subject,
		"iat": nbf,
		"nbf": nbf,
		"exp": exp,
	}

	jwt, err := s.signJWT(claims)
	if err != nil {
		return Token{}, err
	}

	return Token{
		AccessToken: jwt,
		TokenType:   "Bearer",
		ExpiresIn:   int(lifetime.Seconds()),
		ExtExpires:  int(lifetime.Seconds()),
		ExpiresOn:   strconv.FormatInt(exp, 10),
		NotBefore:   strconv.FormatInt(nbf, 10),
	}, nil
}

func (s *Service) signJWT(claims map[string]any) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := strings.Join([]string{encodedHeader, encodedClaims}, ".")

	mac := hmac.New(sha256.New, []byte(s.cfg.TokenKey))
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return strings.Join([]string{unsigned, signature}, "."), nil
}
