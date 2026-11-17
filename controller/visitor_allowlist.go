package controller

import (
	"encoding/json"
	"net/http"
	"net/netip"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
	"go-fly-muti/ws"
	"golang.org/x/text/language"
)

func GetVisitorAllowlist(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		entID := ""
		if scope == "merchant" {
			entID = c.GetString("ent_id")
		}
		policy, err := models.GetVisitorAllowlist(entID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "读取 IP 白名单失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "result": gin.H{"policy": policy, "region_codes": visitorRegionCodes()}})
	}
}

func visitorRegionCodes() []string {
	var codes []string
	for first := byte('A'); first <= 'Z'; first++ {
		for second := byte('A'); second <= 'Z'; second++ {
			code := string([]byte{first, second})
			region, err := language.ParseRegion(code)
			if err == nil && region.String() == code && region.IsCountry() {
				codes = append(codes, code)
			}
		}
	}
	return codes
}

func PostVisitorAllowlist(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
		var policy models.VisitorAllowlist
		if c.ShouldBindJSON(&policy) != nil || !normalizeVisitorAllowlist(&policy) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "请输入有效的单独 IP 或地区。启用时至少添加一项"})
			return
		}
		raw, _ := json.Marshal(policy)
		if len(raw) > 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "白名单项目过多"})
			return
		}
		const name = "访客 IP 白名单"
		entID := ""
		var err error
		if scope == "merchant" {
			entID = c.GetString("ent_id")
			err = models.SaveEntConfig(entID, name, models.VisitorAllowlistKey, string(raw))
		} else {
			err = models.SaveConfig(name, models.VisitorAllowlistKey, string(raw))
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存 IP 白名单失败"})
			return
		}
		ws.CloseVisitorsOutsideAllowlist(entID)
		c.JSON(http.StatusOK, gin.H{"code": 200, "result": policy})
	}
}

func normalizeVisitorAllowlist(policy *models.VisitorAllowlist) bool {
	for i, value := range policy.IPs {
		ip, err := netip.ParseAddr(strings.TrimSpace(value))
		if err != nil || ip.Zone() != "" || ip.IsUnspecified() {
			return false
		}
		policy.IPs[i] = ip.Unmap().String()
	}
	for i, value := range policy.Regions {
		code := strings.ToUpper(strings.TrimSpace(value))
		region, err := language.ParseRegion(code)
		if err != nil || len(code) != 2 || region.String() != code || !region.IsCountry() {
			return false
		}
		policy.Regions[i] = code
	}
	slices.Sort(policy.IPs)
	policy.IPs = slices.Compact(policy.IPs)
	slices.Sort(policy.Regions)
	policy.Regions = slices.Compact(policy.Regions)
	return !policy.Enabled || len(policy.IPs)+len(policy.Regions) > 0
}
