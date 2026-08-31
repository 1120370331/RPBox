package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func Init(secret string) {
	jwtSecret = []byte(secret)
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type SwitchClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Purpose  string `json:"purpose"`
	jwt.RegisteredClaims
}

const accountSwitchPurpose = "account_switch"

type bearerTokenClaims struct {
	Claims
	Purpose string `json:"purpose,omitempty"`
}

func GenerateToken(userID uint, username string, expireHours int) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func GenerateSwitchToken(userID uint, username string, expireDays int) (string, time.Time, error) {
	expiresAt := time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)
	claims := SwitchClaims{
		UserID:   userID,
		Username: username,
		Purpose:  accountSwitchPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	return tokenString, expiresAt, err
}

func ParseToken(tokenStr string) (*Claims, error) {
	parsedClaims := &bearerTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, parsedClaims, jwtKey,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if token.Valid && parsedClaims.Purpose == "" {
		return &parsedClaims.Claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}

func ParseSwitchToken(tokenStr string) (*SwitchClaims, error) {
	parsedClaims := &SwitchClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, parsedClaims, jwtKey,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if token.Valid && parsedClaims.Purpose == accountSwitchPurpose {
		return parsedClaims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}

func jwtKey(token *jwt.Token) (interface{}, error) {
	if token.Method != jwt.SigningMethodHS256 {
		return nil, jwt.ErrSignatureInvalid
	}
	return jwtSecret, nil
}
