package controller

import (
	"fmt"
	"strconv"

	"go-fly-muti/models"
)

func userBelongsToEnterprise(user models.User, entID string) bool {
	return strconv.FormatUint(uint64(user.ID), 10) == entID ||
		strconv.FormatUint(uint64(user.Pid), 10) == entID
}

func validateKefuConversation(entID, kefuID string, visitor models.Visitor) error {
	if visitor.ID == 0 {
		return fmt.Errorf("访客不存在")
	}
	if visitor.EntId != entID {
		return fmt.Errorf("访客不属于当前企业")
	}
	if visitor.ToId != kefuID {
		return fmt.Errorf("会话已转接给其他客服，请刷新会话列表")
	}
	return nil
}
