package tokens

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type TokenService struct {
	db                *sql.DB
	hmacSecret        []byte
	accessTokenExpiry time.Duration
	refreshExpiry     time.Duration
}

func NewTokenService(db *sql.DB, secret string, accessExp, refreshExp time.Duration) *TokenService {
	return &TokenService{
		db:                db,
		hmacSecret:        []byte(secret),
		accessTokenExpiry: accessExp,
		refreshExpiry:     refreshExp,
	}
}

type CustomClaims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

func (c *CustomClaims) Valid() error {
	// Check expiration if set
	if c.ExpiresAt != nil {
		if !c.VerifyExpiresAt(time.Now(), true) {
			return errors.New("token is expired or missing exp claim")
		}
	}
	return nil
}

func (ts *TokenService) GenerateTokens(userID string, roles []string) (string, string, error) {
	// 1) access token
	atClaims := &CustomClaims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ts.accessTokenExpiry)),
			Issuer:    "auth-service",
		},
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	accessToken, err := at.SignedString(ts.hmacSecret)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}
	refreshToken := uuid.NewString()

	return accessToken, refreshToken, nil
}

func (ts *TokenService) StoreRefreshToken(userID, refreshToken string) error {
	rtExpiresAt := time.Now().Add(ts.refreshExpiry)
	return ts.insertRefreshToken(refreshToken, userID, rtExpiresAt)
}

func (ts *TokenService) insertRefreshToken(token, userID string, exp time.Time) error {
	_, err := ts.db.Exec(`
        INSERT INTO refresh_tokens (token, user_id, expires_at)
        VALUES ($1, $2, $3)
    `, token, userID, exp)
	return err
}

func (ts *TokenService) ValidateRefreshToken(refreshToken string) (string, error) {
	var userID string
	var expiresAt time.Time

	row := ts.db.QueryRow(`
        SELECT user_id, expires_at
        FROM refresh_tokens
        WHERE token = $1
    `, refreshToken)

	if err := row.Scan(&userID, &expiresAt); err != nil {
		return "", errors.New("invalid refresh token or not found")
	}
	if time.Now().After(expiresAt) {
		return "", errors.New("refresh token expired")
	}
	return userID, nil
}

func (ts *TokenService) RevokeRefreshToken(refreshToken string) error {
	_, err := ts.db.Exec("DELETE FROM refresh_tokens WHERE token = $1", refreshToken)
	return err
}

