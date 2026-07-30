package tools

import (
	"github.com/golang-jwt/jwt"
	"testing"
	"time"
)

func TestJwt(t *testing.T) {
	tokenCliams := UserClaims{
		Id:         1,
		Username:   "kefu2",
		RoleId:     2,
		Pid:        1,
		CreateTime: time.Now().Format("2006-01-02 15:04:05"),
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Unix() + 24*3600,
		},
	}
	token, err := MakeCliamsToken(tokenCliams)
	t.Log(token, err)

	orgToken, err := ParseCliamsToken(token, true)
	if err != nil {
		t.Fatalf("parse claims token: %v", err)
	}
	if orgToken == nil || orgToken.Username != tokenCliams.Username {
		t.Fatalf("claims = %+v", orgToken)
	}
}

func TestParseTokenRejectsUnexpectedSigningMethod(t *testing.T) {
	t.Setenv("GOFLY_JWT_SECRET", "test-secret")
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{"name": "attacker"})
	tokenString, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	if claims := ParseToken(tokenString); claims != nil {
		t.Fatalf("unexpected claims from HS384 token: %+v", claims)
	}
}
