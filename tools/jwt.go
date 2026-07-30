package tools

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt"
	"go-fly-muti/common"
	"os"
)

func jwtSecret() []byte {
	if secret := os.Getenv("GOFLY_JWT_SECRET"); secret != "" {
		return []byte(secret)
	}
	return []byte(common.JwtSecret)
}

func validateJWTSigningMethod(token *jwt.Token) (interface{}, error) {
	if token.Method != jwt.SigningMethodHS256 {
		return nil, fmt.Errorf("unexpected JWT signing method: %s", token.Method.Alg())
	}
	return jwtSecret(), nil
}

type UserClaims struct {
	Id         uint   `json:"id"`
	Pid        uint   `json:"pid"`
	Username   string `json:"username"`
	RoleName   string `json:"role_name"`
	RoleId     uint   `json:"role_id"`
	CreateTime string `json:"create_time"`
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
