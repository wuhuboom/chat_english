package models

import (
	"time"

	"github.com/jinzhu/gorm"
	"go-fly-muti/types"
)

type Ipblack struct {
	ID       uint       `gorm:"primary_key" json:"id"`
	IP       string     `json:"ip"`
	Name     string     `json:"name"`
	KefuId   string     `json:"kefu_id"`
	EntId    string     `gorm:"type:varchar(100);index" json:"ent_id"`
	CreateAt types.Time `json:"create_at"`
}

func CreateIpblack(ip string, kefuId, entId, name string) (uint, error) {
	black := &Ipblack{
		IP:       ip,
		KefuId:   kefuId,
		EntId:    entId,
		Name:     name,
		CreateAt: types.Time{Time: time.Now()},
	}
	if err := DB.Create(black).Error; err != nil {
		return 0, err
	}
	return black.ID, nil
}

func DeleteIpblackByIp(ip string) error {
	return DB.Where("ip = ?", ip).Delete(&Ipblack{}).Error
}

// DeleteIpblackByIpAndEntId removes only entries owned by the authenticated
// enterprise. The legacy branch keeps pre-ent_id rows manageable by resolving
// their creator against the enterprise account and its child agents.
func DeleteIpblackByIpAndEntId(ip, entId string) error {
	return tenantIpblackScope(DB.Where("ip = ?", ip), entId).Delete(&Ipblack{}).Error
}
func FindIp(ip string) Ipblack {
	var ipblack Ipblack
	DB.Where("ip = ?", ip).First(&ipblack)
	return ipblack
}
func FindIpsByKefuId(id string) []Ipblack {
	var ipblack []Ipblack
	DB.Where("kefu_id = ?", id).Find(&ipblack)
	return ipblack
}

func FindIpsByEntId(entId string) []Ipblack {
	var ipblacks []Ipblack
	tenantIpblackScope(DB, entId).Order("create_at desc").Find(&ipblacks)
	return ipblacks
}

func tenantIpblackScope(db *gorm.DB, entId string) *gorm.DB {
	legacyKefuNames := DB.Model(&User{}).Select("name").Where("id = ? OR pid = ?", entId, entId).SubQuery()
	return db.Where("ent_id = ? OR ((ent_id IS NULL OR ent_id = '') AND kefu_id IN (?))", entId, legacyKefuNames)
}
func FindIps(query interface{}, args []interface{}, page uint, pagesize uint) []Ipblack {
	offset := (page - 1) * pagesize
	if offset < 0 {
		offset = 0
	}
	var ipblacks []Ipblack
	if query != nil {
		DB.Where(query, args...).Offset(offset).Limit(pagesize).Find(&ipblacks)
	} else {
		DB.Offset(offset).Limit(pagesize).Find(&ipblacks)
	}
	return ipblacks
}

// 查询条数
func CountIps(query interface{}, args []interface{}) uint {
	var count uint
	if query != nil {
		DB.Model(&Ipblack{}).Where(query, args...).Count(&count)
	} else {
		DB.Model(&Ipblack{}).Count(&count)
	}
	return count
}
