package models

import (
	"crypto/hmac"
	"crypto/sha1"
	"database/sql"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"go-fly-muti/tools"
	"go-fly-muti/types"
	"os"
	"testing"
	"time"
)

func testAccountCode(secret string) string {
	key, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(time.Now().Unix()/30))
	h := hmac.New(sha1.New, key)
	h.Write(counter[:])
	b := h.Sum(nil)
	o := b[19] & 15
	v := binary.BigEndian.Uint32(b[o:o+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", v%1000000)
}

// Opt-in integration test creates and removes an isolated database, never
// reading or changing customer messages, accounts, or authentication policy.
func TestAccountTOTPIsolationAndEnrollment(t *testing.T) {
	config := os.Getenv("GOFLY_TEST_MYSQL_CONFIG")
	if config == "" {
		t.Skip("set GOFLY_TEST_MYSQL_CONFIG for isolated MySQL integration test")
	}
	raw, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var c types.Mysql
	if err = json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("%s:%s@tcp(%s:%s)/", c.Username, c.Password, c.Server, c.Port)
	root, err := sql.Open("mysql", prefix+"?parseTime=true")
	if err != nil {
		t.Fatal("open database")
	}
	defer root.Close()
	name := fmt.Sprintf("gofly_totp_test_%d", time.Now().UnixNano())
	if _, err = root.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatal("create isolated database:", err)
	}
	defer root.Exec("DROP DATABASE " + name)
	db, err := gorm.Open("mysql", prefix+name+"?parseTime=true")
	if err != nil {
		t.Fatal("connect isolated database")
	}
	defer db.Close()
	db.SingularTable(true)
	previous := DB
	previousConfigs := CustomConfigs
	DB = db
	CustomConfigs = []Config{{ConfKey: GoogleAuthenticatorEnabledKey, ConfValue: "true"}}
	defer func() { DB = previous; CustomConfigs = previousConfigs }()
	if err = db.AutoMigrate(&AccountTOTP{}).Error; err != nil {
		t.Fatal(err)
	}
	a, b := User{ID: 101, Name: "alice"}, User{ID: 102, Name: "bob"}
	bindA, err := CheckAccountTOTP(a, "")
	if err != nil || bindA == nil {
		t.Fatal("alice enrollment not offered", err)
	}
	bindB, err := CheckAccountTOTP(b, "")
	if err != nil || bindB == nil {
		t.Fatal("bob enrollment not offered", err)
	}
	secretA := bindA["secret"].(string)
	secretB := bindB["secret"].(string)
	if secretA == secretB {
		t.Fatal("accounts share secrets")
	}
	if AccountTOTPBound(a.ID) {
		t.Fatal("pending enrollment counted as bound")
	}
	if _, err = CheckAccountTOTP(a, "invalid"); err == nil {
		t.Fatal("invalid enrollment code accepted")
	}
	if bind, err := CheckAccountTOTP(a, testAccountCode(secretA)); err != nil || bind != nil {
		t.Fatal("enrollment confirmation failed", err)
	}
	if !AccountTOTPBound(a.ID) {
		t.Fatal("confirmed enrollment not saved")
	}
	if bind, err := CheckAccountTOTP(a, testAccountCode(secretA)); err != nil || bind != nil {
		t.Fatal("bound account's own code rejected or secret re-disclosed", err)
	}
	if _, err = CheckAccountTOTP(a, ""); err == nil {
		t.Fatal("bound account logged in without code")
	}
	// Choose a code that cannot match any accepted time window for Alice.
	wrong := testAccountCode(secretB)
	for tools.ValidateTOTP(secretA, wrong, time.Now()) {
		secretB, _ = tools.GenerateTOTPSecret()
		wrong = testAccountCode(secretB)
	}
	if _, err = CheckAccountTOTP(a, wrong); err == nil {
		t.Fatal("another account's code accepted")
	}
	db.Model(&AccountTOTP{}).Where("user_id = ?", b.ID).Update("pending_until", time.Now().Unix()-1)
	renewed, err := CheckAccountTOTP(b, testAccountCode(bindB["secret"].(string)))
	if err != nil || renewed == nil || renewed["secret"] == bindB["secret"] {
		t.Fatal("expired enrollment not replaced", err)
	}
	if AccountTOTPBound(b.ID) {
		t.Fatal("expired code completed binding")
	}
	bytes, _ := json.Marshal(AccountTOTP{Secret: secretA, Pending: secretB})
	if string(bytes) != "{}" {
		t.Fatal("credentials serialized")
	}
	CustomConfigs = nil
	if bind, err := CheckAccountTOTP(b, ""); err != nil || bind != nil {
		t.Fatal("disabled policy rejected login", err)
	}
}
