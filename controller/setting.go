package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/silenceper/wechat/v2"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"go-fly-muti/lib"
	"go-fly-muti/models"
	"go-fly-muti/setting"
	"path"
	"strconv"
	"strings"
)

const maxConversationSLAMinutes = 24 * 60
const maxAgentOfflineGraceSeconds = 300

func parseConversationSLASettings(warningValue, overdueValue string) (int, int, error) {
	warningMinutes, err := strconv.Atoi(strings.TrimSpace(warningValue))
	if err != nil || warningMinutes < 1 {
		return 0, 0, fmt.Errorf("预警时间必须是大于0的整数分钟")
	}
	overdueMinutes, err := strconv.Atoi(strings.TrimSpace(overdueValue))
	if err != nil || overdueMinutes <= warningMinutes {
		return 0, 0, fmt.Errorf("超时时间必须大于预警时间")
	}
	if overdueMinutes > maxConversationSLAMinutes {
		return 0, 0, fmt.Errorf("超时时间不能超过24小时")
	}
	return warningMinutes, overdueMinutes, nil
}

func parseRoutingSettings(autoReassignValue, graceValue string) (bool, int, error) {
	autoReassign, err := strconv.ParseBool(strings.TrimSpace(autoReassignValue))
	if err != nil {
		return false, 0, fmt.Errorf("自动转接开关参数无效")
	}
	graceSeconds, err := strconv.Atoi(strings.TrimSpace(graceValue))
	if err != nil || graceSeconds < 3 || graceSeconds > maxAgentOfflineGraceSeconds {
		return false, 0, fmt.Errorf("离线宽限时间必须是3到300秒之间的整数")
	}
	return autoReassign, graceSeconds, nil
}

func GetConfigs(c *gin.Context) {
	configs := models.FindConfigs()
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": configs,
	})
}
func GetEntConfigs(c *gin.Context) {
	entId, _ := c.Get("ent_id")
	configs := models.FindEntConfigs(entId)
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": configs,
	})
}
func GetConfig(c *gin.Context) {
	key := c.Query("key")
	config := models.FindConfig(key)
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": config,
	})
}
func PostEntConfigs(c *gin.Context) {
	kefuId, _ := c.Get("kefu_id")
	name := c.PostForm("name")
	key := c.PostForm("key")
	value := c.PostForm("value")
	if key == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "error",
		})
		return
	}
	config := models.FindEntConfig(kefuId, key)
	if config.ID == 0 {
		models.CreateEntConfig(kefuId, name, key, value)
	} else {
		models.UpdateEntConfig(kefuId, name, key, value)
	}

	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": "",
	})
}

func PostConversationSLASettings(c *gin.Context) {
	entId, _ := c.Get("ent_id")
	warningMinutes, overdueMinutes, err := parseConversationSLASettings(
		c.PostForm("warning_minutes"),
		c.PostForm("overdue_minutes"),
	)
	if err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if err := models.SaveConversationSLASettings(
		fmt.Sprintf("%v", entId),
		strconv.Itoa(warningMinutes),
		strconv.Itoa(overdueMinutes),
	); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "保存SLA设置失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "SLA设置已更新",
		"result": gin.H{
			"warning_minutes": warningMinutes,
			"overdue_minutes": overdueMinutes,
		},
	})
}

func PostRoutingSettings(c *gin.Context) {
	entId, _ := c.Get("ent_id")
	autoReassign, graceSeconds, err := parseRoutingSettings(
		c.PostForm("auto_reassign"),
		c.PostForm("offline_grace_seconds"),
	)
	if err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if err := models.SaveRoutingSettings(
		fmt.Sprintf("%v", entId),
		strconv.FormatBool(autoReassign),
		strconv.Itoa(graceSeconds),
	); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "保存分配设置失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "客服分配设置已更新",
		"result": gin.H{
			"auto_reassign":         autoReassign,
			"offline_grace_seconds": graceSeconds,
		},
	})
}

// 保存微信菜单数据
func PostWechatMenu(c *gin.Context) {
	kefuId, _ := c.Get("kefu_id")
	menu := c.PostForm("menu")
	name := c.PostForm("name")
	if menu == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "error",
		})
		return
	}
	config := models.FindEntConfig(kefuId, "WechatMenu")
	if config.ID == 0 {
		models.CreateEntConfig(kefuId, name, "WechatMenu", menu)
	} else {
		models.UpdateEntConfig(kefuId, name, "WechatMenu", menu)
	}

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
	})
}

// 生成微信菜单
func GetWechatMenu(c *gin.Context) {
	entId, _ := c.Get("ent_id")
	config := models.FindEntConfig(entId, "WechatMenu")
	if config.ID == 0 || config.ConfValue == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "没有菜单数据",
		})
		return
	}
	wechatConfig, err := lib.NewWechatLib(entId.(string))
	if wechatConfig == nil {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	wc := wechat.NewWechat()
	cfg := &offConfig.Config{
		AppID:     wechatConfig.AppId,
		AppSecret: wechatConfig.AppSecret,
		Token:     wechatConfig.Token,
		//EncodingAESKey: "xxxx",
		Cache: memory,
	}
	officialAccount := wc.GetOfficialAccount(cfg)
	menu := officialAccount.GetMenu()
	err = menu.SetMenuByJSON(config.ConfValue)
	if err != nil {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
	})
}
func PostConfig(c *gin.Context) {
	key := c.PostForm("key")
	value := c.PostForm("value")
	if key == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "error",
		})
		return
	}
	if key == "SystemTimezone" {
		if err := setting.ConfigureTimezone(value); err != nil {
			c.JSON(200, gin.H{
				"code": 400,
				"msg":  err.Error(),
			})
			return
		}
	}
	if key == setting.H5ChatTemplateConfigKey {
		template, err := setting.ValidateH5ChatTemplate(value)
		if err != nil {
			c.JSON(200, gin.H{
				"code": 400,
				"msg":  err.Error(),
			})
			return
		}
		value = template
	}
	name := ""
	if key == "SystemTimezone" {
		name = "系统时区"
	}
	if key == setting.H5ChatTemplateConfigKey {
		name = "H5访客界面"
	}
	if err := models.SaveConfig(name, key, value); err != nil {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "保存配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "保存成功",
	})
}

// 上传微信认证文件
func PostUploadWechatFile(c *gin.Context) {
	SendAttachment, err := strconv.ParseBool(models.FindConfig("SendAttachment"))
	if !SendAttachment || err != nil {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "禁止上传附件!",
		})
		return
	}
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "上传失败!",
		})
		return
	} else {

		fileExt := strings.ToLower(path.Ext(f.Filename))
		if f.Size >= 1*1024*1024 {
			c.JSON(200, gin.H{
				"code": 400,
				"msg":  "上传失败!不允许超过1M",
			})
			return
		}
		if fileExt != ".txt" {
			c.JSON(200, gin.H{
				"code": 400,
				"msg":  "上传失败!只允许txt文件",
			})
			return
		}

		c.SaveUploadedFile(f, f.Filename)
		c.JSON(200, gin.H{
			"code": 200,
			"msg":  "上传成功!",
			"result": gin.H{
				"path": "/" + f.Filename,
			},
		})
	}
}
