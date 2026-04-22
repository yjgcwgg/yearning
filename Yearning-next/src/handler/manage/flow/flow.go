package flow

import (
	"Yearning-go/src/handler/common"
	"Yearning-go/src/i18n"
	"Yearning-go/src/model"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cookieY/yee"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

func TplGetAPis(c yee.Context) (err error) {
	switch c.QueryParam("tp") {
	case "user":
		var user []model.CoreAccount
		model.DB().Select("username,real_name").Find(&user)
		return c.JSON(http.StatusOK, common.SuccessPayload(user))
	case "flow":
		var flows []model.CoreWorkflowTpl
		model.DB().Model(model.CoreWorkflowTpl{}).Find(&flows)
		return c.JSON(http.StatusOK, common.SuccessPayload(flows))
	default:
		return
	}
}

func TplPostSourceTemplate(c yee.Context) (err error) {
	u := new(tplTypes)
	if err = c.Bind(u); err != nil {
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_BIND)))
	}
	var t model.CoreWorkflowTpl
	step, _ := json.Marshal(u.Steps)
	if err := model.DB().Where("id =?", u.ID).First(&t).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		model.DB().Create(&model.CoreWorkflowTpl{Source: u.Source, Steps: step})
	} else {
		undo := checkFlowOrderCompletion()
		if len(undo) == 0 {
			model.DB().Model(model.CoreWorkflowTpl{}).Where("id =?", u.ID).Updates(model.CoreWorkflowTpl{Source: u.Source, Steps: step})
		} else {
			return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(fmt.Sprintf(i18n.DefaultLang.Load(i18n.ERR_FLOW_ORDER_IS_NOT_COMPLETE), strings.Join(undo, " "))))
		}
	}

	return c.JSON(http.StatusOK, common.SuccessPayLoadToMessage(i18n.DefaultLang.Load(i18n.INFO_DATA_IS_UPDATED)))
}

func EditSourceTemplateInfo(c yee.Context) (err error) {
	u := new(tplTypes)
	if err = c.Bind(u); err != nil {
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_BIND)))
	}
	var t model.CoreWorkflowTpl
	model.DB().Where("id =?", u.ID).First(&t)
	return c.JSON(http.StatusOK, common.SuccessPayload(t))
}

func DeleteSourceTemplateInfo(c yee.Context) (err error) {
	id := c.QueryParam("id")
	model.DB().Model(model.CoreWorkflowTpl{}).Where("id =?", id).Delete(&model.CoreWorkflowTpl{})
	model.DB().Model(model.CoreDataSource{}).Where("flow_id =?", id).Updates(&model.CoreDataSource{FlowID: -1})
	return c.JSON(http.StatusOK, common.SuccessPayLoadToMessage(i18n.DefaultLang.Load(i18n.INFO_DATA_IS_DELETE)))
}
