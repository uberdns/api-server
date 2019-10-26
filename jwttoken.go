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
}

type JWTTokens struct {
	AccessToken  JWTToken
	RefreshToken JWTToken
}

func (t *JWTToken) New(tokenType string, userId int) {
	type Claim struct {
		TokenType string `json:"token_type"`
		UserID    int    `json:"user_id"`
		JTI       string `json:"jti"`

		jwt.StandardClaims
	}
	if t.ExpiresAt == 0 {
		t.ExpiresAt = time.Now().Unix() + int64(60*5)
	}

	jti, err := uuid.NewUUID()
	if err != nil {
		log.Fatal(err)
	}

	claims := Claim{
		tokenType,
		userId,
		jti.String(),
		jwt.StandardClaims{
			ExpiresAt: t.ExpiresAt,
			Issuer:    t.Issuer,
		},
	}

	t.Token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
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
	token.Salt = "bj#zojhb&my%6lcs$7t5w)hzb@7s-mhxvqd35h9##f%kywo%$7"
	token.ExpiresAt = time.Now().Unix() + int64(60*5)
	token.New("access", userId)
	j.AccessToken = token
}

func (j *JWTTokens) NewRefresh(userId int) {
	token := JWTToken{}
	token.Salt = "bj#zojhb&my%6lcs$7t5w)hzb@7s-mhxvqd35h9##f%kywo%$7"
	token.ExpiresAt = time.Now().Unix() + int64(60*10)
	token.New("refresh", userId)
	j.RefreshToken = token
}

func (j *JWTTokens) New(userId int) {
	j.NewAccess(userId)
	j.NewRefresh(userId)
}

func (j *JWTTokens) String() string {
	type TokenMessage struct {
		Access  string `json:"access"`
		Refresh string `json:"refresh"`
	}
	tokenMSG := TokenMessage{}
	tokenMSG.Access = j.AccessToken.String()
	tokenMSG.Refresh = j.RefreshToken.String()

	jsonMSG, err := json.Marshal(tokenMSG)
	if err != nil {
		log.Fatal(err)
	}

	return string(jsonMSG)
}
