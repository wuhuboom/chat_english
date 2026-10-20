package tools

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt"
	"log"
	"os"
	"strings"
	"sync"
)

var (
	jwtSecretOnce sync.Once
	jwtSecretData []byte
)

func jwtSecret() []byte {
	jwtSecretOnce.Do(func() {
		secret := strings.TrimSpace(os.Getenv("GOFLY_JWT_SECRET"))
		if len(secret) >= 32 {
			jwtSecretData = []byte(secret)
			return
		}
		// Never fall back to the old public, hard-coded signing key.  A random
		// per-process key is safer and deliberately invalidates all sessions on
		// restart. Production deployments must set GOFLY_JWT_SECRET so sessions
		// remain valid across restarts and multiple instances.
		random := make([]byte, 64)
		if _, err := rand.Read(random); err != nil {
			panic(fmt.Sprintf("cannot generate JWT signing secret: %v", err))
		}
		jwtSecretData = []byte(hex.EncodeToString(random))
		log.Print("SECURITY WARNING: GOFLY_JWT_SECRET is missing or shorter than 32 characters; using an ephemeral random key")
	})
	return jwtSecretData
}

func validateJWTSigningMethod(token *jwt.Token) (interface{}, error) {
	if token.Method != jwt.SigningMethodHS256 {
		return nil, fmt.Errorf("unexpected JWT signing method: %s", token.Method.Alg())
	}
	return jwtSecret(), nil
}

type UserClaims struct {
	MFAVerified bool   `json:"mfa_verified"`
	Id          uint   `json:"id"`
	Pid         uint   `json:"pid"`
	Username    string `json:"username"`
	RoleName    string `json:"role_name"`
	RoleId      uint   `json:"role_id"`
	CreateTime  string `json:"create_time"`
	AuthVersion int64  `json:"auth_version"`
	jwt.StandardClaims
}

func MakeToken(obj map[string]interface{}) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(obj))
	tokenString, err := token.SignedString(jwtSecret())
	return tokenString, err
}
func ParseToken(tokenStr string) map[string]interface{} {
	token, err := jwt.Parse(tokenStr, validateJWTSigningMethod)

	if err != nil || token == nil || !token.Valid {
		return nil
	}
	finToken, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil
	}
	return finToken
}

/*
*
生成jwt
*/
func MakeCliamsToken(obj UserClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, obj)
	tokenString, err := token.SignedString(jwtSecret())
	return tokenString, err
}

/*
*
解析jwt token
*/
func ParseCliamsToken(token string, validExpired bool) (*UserClaims, error) {
	if token == "" {
		return nil, errors.New("token failed")
	}
	tokenClaims, err := jwt.ParseWithClaims(token, &UserClaims{}, validateJWTSigningMethod)

	if err != nil || tokenClaims == nil {
		return nil, err
	}
	claims, ok := tokenClaims.Claims.(*UserClaims)
	if !ok {
		return nil, errors.New("token claims failed")
	}
	if validExpired && !tokenClaims.Valid {
		return nil, err
	}
	return claims, nil
}
