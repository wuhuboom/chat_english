package middleware

import (
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
	"go-fly-muti/setting"
	"go-fly-muti/tools"
)

func VisitorIPAllowed(c *gin.Context, entID string) bool {
	if VisitorIPMatches(setting.VisitorClientIP(c.Request), entID) {
		return true
	}
	RejectVisitorAccess(c)
	return false
}

func VisitorConversationIPAllowed(c *gin.Context, entID, visitorID string) bool {
	if visitorID != "" {
		visitor := models.FindVisitorByVistorId(visitorID)
		if entID == "" {
			entID = visitor.EntId
		} else if visitor.EntId != "" && visitor.EntId != entID && !VisitorIPAllowed(c, visitor.EntId) {
			return false
		}
	}
	return VisitorIPAllowed(c, entID)
}

func VisitorLinkIPAllowed(c *gin.Context, entID, kefuName string) bool {
	if entID == "" && kefuName != "" {
		agent := models.FindUser(kefuName)
		if agent.Pid != 0 {
			entID = strconv.FormatUint(uint64(agent.Pid), 10)
		} else if agent.ID != 0 {
			entID = strconv.FormatUint(uint64(agent.ID), 10)
		}
	}
	return VisitorIPAllowed(c, entID)
}

// VisitorIPMatches applies the global policy and then the visitor's own merchant policy.
func VisitorIPMatches(ip netip.Addr, entID string) bool {
	policies := make([]models.VisitorAllowlist, 0, 2)
	global, err := models.GetVisitorAllowlist("")
	if err != nil {
		return false
	}
	policies = append(policies, global)
	if entID != "" {
		id, err := strconv.ParseUint(strings.TrimSpace(entID), 10, 32)
		if err != nil || id == 0 {
			return false
		}
		entID = strconv.FormatUint(id, 10)
		merchant, err := models.GetVisitorAllowlist(entID)
		if err != nil {
			return false
		}
		policies = append(policies, merchant)
	}
	region := ""
	regionChecked := false
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}
		if !ip.IsValid() || ip.IsUnspecified() {
			return false
		}
		if slices.Contains(policy.IPs, ip.String()) {
			continue
		}
		if len(policy.Regions) == 0 {
			return false
		}
		if !regionChecked {
			region = tools.IPRegionCode(net.IP(ip.AsSlice()))
			regionChecked = true
		}
		if region == "" || !slices.Contains(policy.Regions, region) {
			return false
		}
	}
	return true
}
