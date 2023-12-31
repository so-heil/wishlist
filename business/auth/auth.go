package auth

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/so-heil/wishlist/business/keystore"
)

const issuer = "Wishlist API"

type key string

const userIDKey = key("user-id-key")

var ErrInvalidToken = errors.New("token provided is not valid")
var ErrMalformedToken = errors.New("expected authorization header format: Bearer <token>")
var ErrNoUserInContext = errors.New("no user attached to context")

type Auth struct {
	signingMethod jwt.SigningMethod
	ks            *keystore.KeyStore
}

func New(ks *keystore.KeyStore) *Auth {
	return &Auth{ks: ks, signingMethod: ks.SigningMethod}
}

// Token creates and signs a JWT token with the most recent key from keystore
func (a *Auth) Token(claims jwt.Claims) (string, error) {
	tk := jwt.NewWithClaims(a.signingMethod, claims)
	uid, key, err := a.ks.Active()
	if err != nil {
		return "", fmt.Errorf("get keystore active key: %w", err)
	}
	tk.Header["kid"] = uid
	tkStr, err := tk.SignedString(key.Signer)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return tkStr, nil
}

// Parse validates and parses the JWT token into the passed claims
func (a *Auth) Parse(token string, dst jwt.Claims) error {
	_, err := jwt.ParseWithClaims(token, dst, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, ErrInvalidToken
		}
		sig, err := a.ks.Signer(kid)
		if err != nil {
			return nil, fmt.Errorf("get signer for kid: %w", err)
		}
		return sig.Signer.Public(), nil
	})
	return err
}

// ParseFromBearer validates and parses a bearer token passed into it, then tries to validate and parse the data from JWT via Parse
// Its a helper function around Parse
func (a *Auth) ParseFromBearer(bearerToken string, dst jwt.Claims) error {
	parts := strings.Split(bearerToken, " ")
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
		return ErrMalformedToken
	}

	return a.Parse(parts[1], dst)
}
