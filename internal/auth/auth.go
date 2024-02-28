package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/beevik/guid"
	"github.com/golang-jwt/jwt/v4"
	"loyalty-system/internal/models/users"
	"strings"
	"time"
)

type AuthorizingData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (d *AuthorizingData) GenerateHashPassword() string {
	hashSum := GetHash(d.Password, SecretKey)

	return hashSum
}

func (d *AuthorizingData) NewUserFromData() *users.User {
	user := &users.User{
		ID:               guid.NewString(),
		Login:            d.Login,
		HashPassword:     d.GenerateHashPassword(),
		RegistrationDate: time.Now(),
	}

	return user
}

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

func BuildJWTString(user *users.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		UserID: user.ID,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GetUserIDFromAuthHeader(header string) (string, error) {
	tokenString := strings.TrimPrefix(header, "Bearer ")

	if tokenString == "" {
		return "", errors.New("empty authorization header")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(SecretKey), nil
		})

	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", fmt.Errorf("token is not valid")
	}

	return claims.UserID, nil
}

func GetHash(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	result := h.Sum(nil)

	return hex.EncodeToString(result)
}

const TokenExp = time.Hour * 3
const SecretKey = "dswereGsdfgert2345Dsd"
