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
	hashSum := GetHash(d.Password, SECRET_KEY)

	return hashSum
}

func (d *AuthorizingData) NewUserFromData() *users.User {
	user := &users.User{
		Id:               guid.NewString(),
		Login:            d.Login,
		HashPassword:     d.GenerateHashPassword(),
		RegistrationDate: time.Now(),
	}

	return user
}

type Claims struct {
	jwt.RegisteredClaims
	UserId string
}

func BuildJWTString(user *users.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserId: user.Id,
	})

	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GetUserIdFromAuthHeader(header string) (string, error) {
	tokenString := strings.Trim(header, `Bearer`)
	tokenString = strings.TrimSpace(tokenString)

	if tokenString == "" {
		return "", errors.New("empty authorization header")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(SECRET_KEY), nil
		})

	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", fmt.Errorf("token is not valid")
	}

	return claims.UserId, nil
}

func GetHash(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	result := h.Sum(nil)

	return hex.EncodeToString(result)
}

const TOKEN_EXP = time.Hour * 3
const SECRET_KEY = "dswereGsdfgert2345Dsd"
