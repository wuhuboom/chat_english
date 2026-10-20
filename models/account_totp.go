package models

import (
	"encoding/base64"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/skip2/go-qrcode"
	"go-fly-muti/tools"
	"time"
)

// AccountTOTP never serializes credentials through account-list APIs.
type AccountTOTP struct {
	UserID       uint   `gorm:"primary_key" json:"-"`
	Secret       string `gorm:"size:64;not null;default:''" json:"-"`
	Pending      string `gorm:"size:64;not null;default:''" json:"-"`
	PendingUntil int64  `json:"-"`
}

func TOTPBinding(secret, account string) (map[string]interface{}, error) {
	issuer := FindConfig("SystemTitle")
	if issuer == "" {
		issuer = "GOFLY"
	}
	png, err := qrcode.Encode(tools.TOTPURI(secret, issuer, account), qrcode.Medium, 256)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"secret": secret, "qr_code": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)}, nil
}

func AccountTOTPBound(id uint) bool {
	var row AccountTOTP
	return DB.Where("user_id = ?", id).First(&row).Error == nil && row.Secret != ""
}

// CheckAccountTOTP runs only after password/status checks. Enrollment never
// issues a login token until possession of the pending secret is demonstrated.
func CheckAccountTOTP(user User, code string) (map[string]interface{}, error) {
	if !GoogleAuthenticatorEnabled() {
		return nil, nil
	}
	var row AccountTOTP
	err := DB.Where("user_id = ?", user.ID).First(&row).Error
	if err != nil {
		if !gorm.IsRecordNotFoundError(err) {
			return nil, err
		}
		if err = DB.Exec("INSERT IGNORE INTO account_totp (user_id, secret, pending, pending_until) VALUES (?, '', '', 0)", user.ID).Error; err != nil {
			return nil, err
		}
		if err = DB.Where("user_id = ?", user.ID).First(&row).Error; err != nil {
			return nil, err
		}
	}
	if row.Secret != "" {
		if !tools.ValidateTOTP(row.Secret, code, time.Now()) {
			return nil, fmt.Errorf("动态验证码错误")
		}
		return nil, nil
	}
	now := time.Now().Unix()
	if row.Pending == "" || row.PendingUntil <= now {
		secret, err := tools.GenerateTOTPSecret()
		if err != nil {
			return nil, err
		}
		result := DB.Model(&AccountTOTP{}).Where("user_id = ? AND secret = '' AND (pending = '' OR pending_until <= ?)", user.ID, now).Updates(map[string]interface{}{"pending": secret, "pending_until": now + 600})
		if result.Error != nil {
			return nil, result.Error
		}
		if err = DB.Where("user_id = ?", user.ID).First(&row).Error; err != nil {
			return nil, err
		}
		// Another request may have completed binding while this request waited.
		if row.Secret != "" {
			return nil, fmt.Errorf("绑定状态已变化，请重新登录")
		}
	} else if code != "" {
		if !tools.ValidateTOTP(row.Pending, code, time.Now()) {
			return nil, fmt.Errorf("动态验证码错误")
		}
		result := DB.Model(&AccountTOTP{}).Where("user_id = ? AND secret = '' AND pending = ? AND pending_until > ?", user.ID, row.Pending, now).Updates(map[string]interface{}{"secret": row.Pending, "pending": "", "pending_until": 0})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected != 1 {
			return nil, fmt.Errorf("绑定已过期或状态已变化，请重试")
		}
		return nil, nil
	}
	return TOTPBinding(row.Pending, user.Name)
}
