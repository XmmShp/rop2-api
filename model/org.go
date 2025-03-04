package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ConfigUseDeparts uint8

const (
	//不显示部门
	DisableDeparts ConfigUseDeparts = 0b0000_0000
	//开启部门机制，只使用统一考核
	UnifiedAssess ConfigUseDeparts = 0b0000_0011
	//开启部门机制，只使用独立考核
	IndependentAssess ConfigUseDeparts = 0b0000_0101
	//开启部门机制，使用独立和统一考核
	MixedAssess ConfigUseDeparts = 0b0000_0111
)

type Org struct {
	Id   uint32 `json:"id" gorm:"primaryKey;autoIncrement;<-:false"` //主键，自动递增
	Name string `json:"name" gorm:"type:varchar(80);not null;unique"`

	CreateAt time.Time `json:"createAt" gorm:"not null;autoCreateTime"`

	DefaultDepart uint32 `json:"defaultDepart" gorm:"uniqueIndex"`

	//组织是否启用部门机制
	UseDeparts ConfigUseDeparts `json:"useDeparts" gorm:"not null;default:0"`
}

func GetOrg(id uint32) *Org {
	var pobj = &Org{}
	result := db.First(pobj, id)
	if result.Error != nil {
		return nil
	} else {
		return pobj
	}
}

func CountOrg() int64 {
	var result int64
	rs := db.Table("orgs").Count(&result)
	if rs.Error != nil {
		fmt.Printf("CountOrg error: %s\r\n", rs.Error.Error())
		return 0
	}
	return result
}

// 内部方法。初始化一个组织（包括默认部门、管理员、默认拒信）。
func InitNewOrg(name string, adminZjuId string, adminNickname string) (newOrgId uint32, funcError error) {
	transactionFunc := func(tx *gorm.DB) (err error) {
		org := &Org{
			Name: name,
		}
		if err = tx.Select("Name").Create(org).Error; err != nil {
			return err
		}
		newOrgId = org.Id
		defaultDepart := &Depart{
			Name:  "默认部门",
			Owner: org.Id,
		}
		if err = tx.Select("Name", "Owner").Create(defaultDepart).Error; err != nil {
			return err
		}
		org.DefaultDepart = defaultDepart.Id
		if err = tx.Select("DefaultDepart").Save(org).Error; err != nil {
			return err
		}
		admin := &Admin{
			ZjuId:    adminZjuId,
			At:       org.Id,
			Nickname: adminNickname,
			Level:    Maintainer,
		}
		if err = tx.Select("ZjuId", "At", "Nickname", "Level").Create(admin).Error; err != nil {
			return err
		}
		//一些默认文本可能需要修改
		defaultRejectTemplate, defaultAcceptTemplate :=
			&Template{
				Owner:   org.Id,
				Name:    "默认拒信",
				Content: "很遗憾，您未能成功加入{组织}。",
			},
			&Template{
				Owner:   org.Id,
				Name:    "默认录取通知",
				Content: "感谢您参与{表单}，您已成功加入{组织}。",
			}
		if err = tx.Select("Owner", "Name", "Content").Create(defaultRejectTemplate).Error; err != nil {
			return err
		}
		if err = tx.Select("Owner", "Name", "Content").Create(defaultAcceptTemplate).Error; err != nil {
			return err
		}
		defaultRejectStage, defaultAcceptStage :=
			&Stage{
				Owner:   org.DefaultDepart,
				Step:    Rejected,
				OnEnter: &defaultRejectTemplate.Id,
			},
			&Stage{
				Owner:   org.DefaultDepart,
				Step:    Accepted,
				OnEnter: &defaultAcceptTemplate.Id,
			}
		if err = tx.Select("Owner", "Step", "OnEnter").Create(defaultRejectStage).Error; err != nil {
			return err
		}
		if err = tx.Select("Owner", "Step", "OnEnter").Create(defaultAcceptStage).Error; err != nil {
			return err
		}

		//成功完成，没有错误
		return
	}
	funcError = db.Transaction(func(tx *gorm.DB) error {
		err := transactionFunc(tx)
		//或许可以处理一下err
		return err
	})
	return
}
