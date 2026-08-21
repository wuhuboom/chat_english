package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go-fly-muti/common"
	"go-fly-muti/models"
)

var createIpblack = models.CreateIpblack
var deleteIpblackByIp = models.DeleteIpblackByIp
var deleteIpblackByIpAndEntId = models.DeleteIpblackByIpAndEntId
var findIpsByEntId = models.FindIpsByEntId
var findAllIps = models.FindIps
var findVisitorsByEntIP = func(entID, ip string) []models.Visitor {
	return models.FindVisitorsByCondition(
		"ent_id = ? AND (client_ip = ? OR source_ip = ?)",
		entID,
		ip,
		ip,
	)
}

func isSuperAdmin(c *gin.Context) bool {
	roleId, exists := c.Get("role_id")
	if !exists {
		return false
	}
	switch value := roleId.(type) {
	case float64:
		return value == 1
	case int:
		return value == 1
	case uint:
		return value == 1
	case string:
		return value == "1"
	default:
		return false
	}
}

func PostIpblack(c *gin.Context) {
	ip := c.PostForm("ip")
	name := c.PostForm("name")
	if ip == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "请输入IP!",
		})
		return
	}
	kefuId := c.GetString("kefu_name")
	entId := c.GetString("ent_id")
	if kefuId == "" || entId == "" {
		c.JSON(200, gin.H{
			"code": 403,
			"msg":  "无法确认当前商户身份",
		})
		return
	}
	if _, err := createIpblack(ip, kefuId, entId, name); err != nil {
		c.JSON(200, gin.H{
			"code": 500,
			"msg":  "添加IP黑名单失败",
		})
		return
	}
	cleanupBlacklistedVisitorsFn(
		entId,
		kefuId,
		"加入IP黑名单，自动结束会话",
		findVisitorsByEntIP(entId, ip),
	)
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "已加入IP黑名单，可在右侧或设置中的IP黑名单解除",
	})
}
func DelIpblack(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "请输入IP!",
		})
		return
	}
	var err error
	if isSuperAdmin(c) {
		err = deleteIpblackByIp(ip)
	} else {
		entId := c.GetString("ent_id")
		if entId == "" {
			c.JSON(200, gin.H{
				"code": 403,
				"msg":  "无法确认当前商户身份",
			})
			return
		}
		err = deleteIpblackByIpAndEntId(ip, entId)
	}
	if err != nil {
		c.JSON(200, gin.H{
			"code": 500,
			"msg":  "解除IP黑名单失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "IP已从黑名单解除",
	})
}
func GetIpblacks(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page == 0 {
		page = 1
	}
	count := models.CountIps(nil, nil)
	list := findAllIps(nil, nil, uint(page), common.VisitorPageSize)
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
		"result": gin.H{
			"list":     list,
			"count":    count,
			"pagesize": common.PageSize,
		},
	})
}
func GetIpblacksByKefuId(c *gin.Context) {
	var list []models.Ipblack
	if isSuperAdmin(c) {
		list = findAllIps(nil, nil, 1, 1000)
	} else {
		entId := c.GetString("ent_id")
		if entId == "" {
			c.JSON(200, gin.H{
				"code": 403,
				"msg":  "无法确认当前商户身份",
			})
			return
		}
		list = findIpsByEntId(entId)
	}
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": list,
	})
}
