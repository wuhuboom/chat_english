package models

type User_role struct {
	ID     uint `gorm:"primary_key" json:"id"`
	UserId uint `json:"user_id"`
	RoleId uint `json:"role_id"`
}

func FindRoleByUserId(userId interface{}) User_role {
	var uRole User_role
	DB.Where("user_id = ?", userId).First(&uRole)
	return uRole
}
func CreateUserRole(userId uint, roleId uint) {
	if userId == 0 || roleId == 0 || DB == nil {
		return
	}

	tx := DB.Begin()
	if tx.Error != nil {
		return
	}

	var user User
	if err := tx.Raw("SELECT id FROM user WHERE id = ? FOR UPDATE", userId).Scan(&user).Error; err != nil || user.ID == 0 {
		tx.Rollback()
		return
	}

	var roles []User_role
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("user_id = ?", userId).Order("id asc").Find(&roles).Error; err != nil {
		tx.Rollback()
		return
	}

	if len(roles) == 0 {
		if err := tx.Create(&User_role{UserId: userId, RoleId: roleId}).Error; err != nil {
			tx.Rollback()
			return
		}
	} else {
		primaryRole := roles[0]
		if err := tx.Model(&primaryRole).Update("role_id", roleId).Error; err != nil {
			tx.Rollback()
			return
		}
		if len(roles) > 1 {
			if err := tx.Where("user_id = ? AND id <> ?", userId, primaryRole.ID).Delete(User_role{}).Error; err != nil {
				tx.Rollback()
				return
			}
		}
	}

	if tx.Commit().Error != nil {
		tx.Rollback()
	}
}
func DeleteRoleByUserId(userId interface{}) {
	DB.Where("user_id = ?", userId).Delete(User_role{})
}
