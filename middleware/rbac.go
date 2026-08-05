package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
	"go-fly-muti/types"
	"strconv"
	"strings"
)

func RbacAuth(c *gin.Context) {
	roleId, _ := c.Get("role_id")
	role := models.FindRole(roleId)
	var flag bool
	rPaths := strings.Split(c.Request.RequestURI, "?")
	uriParam := fmt.Sprintf("%s:%s", c.Request.Method, rPaths[0])
	if role.Method != "*" || role.Path != "*" {
		paths := strings.Split(role.Path, ",")
		for _, p := range paths {
			if uriParam == p {
				flag = true
				break
			}
		}
		if !flag {
			c.JSON(200, gin.H{
				"code": 403,
				"msg":  "没有权限:" + uriParam,
			})
			c.Abort()
			return
		}
		//methods := strings.Split(role.Method, ",")
		//for _, m := range methods {
		//	if c.Request.Method == m {
		//		methodFlag = true
		//		break
		//	}
		//}
		//if !methodFlag {
		//	c.JSON(200, gin.H{
		//		"code": 403,
		//		"msg":  "没有权限:" + c.Request.Method + "," + rPaths[0],
		//	})
		//	c.Abort()
		//	return
		//}
	}
	//var flag bool
	//if role.Path != "*" {
	//	paths := strings.Split(role.Path, ",")
	//	for _, p := range paths {
	//		if rPaths[0] == p {
	//			flag = true
	//			break
	//		}
	//	}
	//	if !flag {
	//		c.JSON(200, gin.H{
	//			"code": 403,
	//			"msg":  "没有权限:" + rPaths[0],
	//		})
	//		c.Abort()
	//		return
	//	}
	//}
}
func AdminAuth(c *gin.Context) {
	roleId, _ := c.Get("role_id")
	if roleId.(float64) != 1 {
		c.JSON(200, gin.H{
			"code": types.ApiCode.NO_ADMIN_AUTH,
			"msg":  types.ApiCode.GetMessage(types.ApiCode.NO_ADMIN_AUTH),
		})
		c.Abort()
		return
	}

}

// MerchantAuth permits only ordinary merchant accounts. Agent accounts must
// not be able to delete enterprise-wide data, and the super administrator has
// a separate system scope rather than impersonating a merchant.
func MerchantAuth(c *gin.Context) {
	roleValue, exists := c.Get("role_id")
	roleID, valid := normalizeRoleID(roleValue)
	if !exists || !valid || roleID != types.Constant.EntRoleId {
		c.JSON(200, gin.H{
			"code": 403,
			"msg":  "仅普通商户管理员可以执行此操作",
		})
		c.Abort()
		return
	}
}

func normalizeRoleID(value interface{}) (uint, bool) {
	switch roleID := value.(type) {
	case float64:
		if roleID < 0 || roleID != float64(uint(roleID)) {
			return 0, false
		}
		return uint(roleID), true
	case uint:
		return roleID, true
	case int:
		if roleID < 0 {
			return 0, false
		}
		return uint(roleID), true
	case string:
		parsed, err := strconv.ParseUint(strings.TrimSpace(roleID), 10, 64)
		return uint(parsed), err == nil
	default:
		return 0, false
	}
}
