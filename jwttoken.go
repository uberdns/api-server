package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

type JWTToken struct {
	Token     *jwt.Token
	Claims    *jwt.Claims
	Salt      string
	Issuer    string
	ExpiresAt int64
	UserID    int
}

type JWTTokens struct {
	AccessToken  JWTToken
	RefreshToken JWTToken
}

type JWTTokenMessage struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

type JWTClaim struct {
	TokenType string `json:"token_type"`
	UserID    int    `json:"user_id"`
	JTI       string `json:"jti"`

	jwt.StandardClaims
}

func (t *JWTToken) New(tokenType string) {

	if t.ExpiresAt == 0 {
		t.ExpiresAt = time.Now().Unix() + int64(60*5)
	}

	jti, err := uuid.NewUUID()
	if err != nil {
		log.Fatal(err)
	}

	claims := JWTClaim{
		tokenType,
		t.UserID,
		jti.String(),
		jwt.StandardClaims{
			ExpiresAt: t.ExpiresAt,
			Issuer:    t.Issuer,
		},
	}

	t.Token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
}

func (t *JWTToken) LookupFromString(tokenStr string) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaim{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(encryptionSalt), nil
	})

	if err != nil {
		log.Fatal(err)
	}

	if claims, ok := token.Claims.(*JWTClaim); ok && token.Valid {
		t.Token = token
		t.UserID = claims.UserID
		t.ExpiresAt = claims.ExpiresAt
		t.Issuer = claims.Issuer
	}
}

func (t *JWTToken) IsValid() bool {
	if time.Now().Unix() < t.ExpiresAt {
		return true
	}

	return false
}

func (t *JWTToken) SetSalt(salt string) {
	t.Salt = salt
}

func (t *JWTToken) String() string {
	sk := []byte(t.Salt)
	token, err := t.Token.SignedString(sk)
	if err != nil {
		log.Fatal(err)
	}
	return token
}

func (j *JWTTokens) NewAccess(userId int) {
	token := JWTToken{}
	token.Salt = encryptionSalt
	token.ExpiresAt = time.Now().Unix() + int64(60*5)
	token.UserID = userId
	token.New("access")
	j.AccessToken = token
}

func (j *JWTTokens) NewRefresh(userId int) {
	token := JWTToken{}
	token.Salt = encryptionSalt
	token.ExpiresAt = time.Now().Unix() + int64(60*10)
	token.UserID = userId
	token.New("refresh")
	j.RefreshToken = token
}

func (j *JWTTokens) New(userId int) {
	j.NewAccess(userId)
	j.NewRefresh(userId)
}

func (j *JWTTokens) String() string {
	tokenMSG := JWTTokenMessage{}
	tokenMSG.Access = j.AccessToken.String()
	tokenMSG.Refresh = j.RefreshToken.String()

	jsonMSG, err := json.Marshal(tokenMSG)
	if err != nil {
		log.Fatal(err)
	}

	return string(jsonMSG)
}
