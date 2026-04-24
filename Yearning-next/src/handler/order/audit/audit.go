package audit

import (
	"Yearning-go/src/handler/common"
	"Yearning-go/src/handler/manage/flow"
	"Yearning-go/src/i18n"
	"Yearning-go/src/lib/calls"
	"Yearning-go/src/lib/factory"
	"Yearning-go/src/lib/pusher"
	"Yearning-go/src/model"
	"encoding/json"
	"github.com/cookieY/yee"
	"github.com/golang-jwt/jwt"
	"golang.org/x/net/websocket"
	"io"
	"net/http"
	"strings"
	"time"
)

const QueryField = "work_id, username, text, backup, date, real_name, `status`, `type`, `delay`, `source`, `source_id`,`id_c`,`data_base`,`table`,`execute_time`,assigned,current_step,relevant"

func AuditOrderState(c yee.Context) (err error) {
	u := new(Confirm)
	user := new(factory.Token).JwtParse(c)
	if err = c.Bind(u); err != nil {
		c.Logger().Error(err.Error())
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_BIND)))
	}

	switch u.Tp {
	case "undo":
		pusher.NewMessagePusher(u.WorkId).Order().OrderBuild(pusher.UndoStatus).Push()
		model.DB().Model(model.CoreSqlOrder{}).Where("work_id =?", u.WorkId).Updates(&model.CoreSqlOrder{Status: 6})
		return c.JSON(http.StatusOK, common.SuccessPayLoadToMessage(i18n.DefaultLang.Load(i18n.INFO_ORDER_IS_UNDO)))
	case "agree":
		return c.JSON(http.StatusOK, MultiAuditOrder(u, user.Username))
	case "reject":
		return c.JSON(http.StatusOK, RejectOrder(u, user.Username))
	default:
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_FAKE)))
	}
}

func BatchAuditOrderState(c yee.Context) (err error) {
	u := new(BatchConfirm)
	user := new(factory.Token).JwtParse(c)
	if err = c.Bind(u); err != nil {
		c.Logger().Error(err.Error())
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_BIND)))
	}

	if len(u.WorkIds) == 0 {
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE("work_ids is empty"))
	}

	result := BatchResult{}

	for _, workId := range u.WorkIds {
		var order model.CoreSqlOrder
		if err := model.DB().Select("source_id, current_step, status").
			Where("work_id = ?", workId).First(&order).Error; err != nil {
			result.Failed = append(result.Failed, BatchFailItem{
				WorkId: workId, Error: "工单不存在",
			})
			continue
		}

		if order.Status != 2 {
			result.Failed = append(result.Failed, BatchFailItem{
				WorkId: workId, Error: "工单状态不是待审核",
			})
			continue
		}

		confirm := &Confirm{
			WorkId:   workId,
			Tp:       u.Tp,
			Text:     u.Text,
			SourceId: order.SourceId,
			Flag:     order.CurrentStep,
		}

		switch u.Tp {
		case "agree":
			resp := MultiAuditOrder(confirm, user.Username)
			if resp.Code != 1200 {
				result.Failed = append(result.Failed, BatchFailItem{
					WorkId: workId, Error: resp.Text,
				})
			} else {
				result.Success = append(result.Success, workId)
			}
		case "reject":
			resp := RejectOrder(confirm, user.Username)
			if resp.Code != 1200 {
				result.Failed = append(result.Failed, BatchFailItem{
					WorkId: workId, Error: resp.Text,
				})
			} else {
				result.Success = append(result.Success, workId)
			}
		default:
			result.Failed = append(result.Failed, BatchFailItem{
				WorkId: workId, Error: "不支持的操作类型",
			})
		}
	}

	return c.JSON(http.StatusOK, common.SuccessPayload(result))
}

func ScheduledChange(c yee.Context) (err error) {
	u := new(Confirm)
	if err = c.Bind(u); err != nil {
		c.Logger().Error(err.Error())
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_BIND)))
	}
	var isCall string
	if client := calls.NewRpc(); client != nil {
		if err := client.Call("Engine.StopDelay", u, &isCall); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, common.SuccessPayLoadToMessage(i18n.DefaultLang.Load(i18n.INFO_ORDER_DELAY_SUCCESS)))
}

// DelayKill will stop delay order
func DelayKill(c yee.Context) (err error) {
	u := new(Confirm)
	if err = c.Bind(u); err != nil {
		c.Logger().Error(err.Error())
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_BIND)))
	}
	user := new(factory.Token).JwtParse(c)
	model.DB().Create(&model.CoreWorkflowDetail{
		WorkId:   u.WorkId,
		Username: user.Username,
		Time:     time.Now().Format("2006-01-02 15:04"),
		Action:   i18n.DefaultLang.Load(i18n.ORDER_KILL_STATE),
	})
	return c.JSON(http.StatusOK, common.SuccessPayLoadToMessage(delayKill(u.WorkId)))
}

func FetchAuditOrder(c yee.Context) (err error) {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		var u common.PageList[[]model.CoreSqlOrder]
		var b []byte
		for {
			if err := websocket.Message.Receive(ws, &b); err != nil {
				if err != io.EOF {
					c.Logger().Error(err)
				}
				break
			}
			if string(b) == "ping" {
				continue
			}
			if err := json.Unmarshal(b, &u); err != nil {
				c.Logger().Error(err)
				break
			}
			token, err := factory.WsTokenParse(ws.Request().Header.Get("Sec-WebSocket-Protocol"))
			if err != nil {
				c.Logger().Error(err)
				break
			}
			user := token.Claims.(jwt.MapClaims)["name"].(string)
			u.Paging().OrderBy("(status = 2) DESC, date DESC").Select(QueryField).Query(common.AccordingToAllOrderState(u.Expr.Status),
				common.AccordingToAllOrderType(u.Expr.Type),
				common.AccordingToRelevant(user),
				common.AccordingToText(u.Expr.Text),
				common.AccordingToUsername(u.Expr.Username),
				common.AccordingToDate(u.Expr.Picker),
				common.AccordingToWorkId(u.Expr.WorkId),
			)
			if err = websocket.Message.Send(ws, factory.ToJson(u.ToMessage())); err != nil {
				c.Logger().Error(err)
				break
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func FetchOSCAPI(c yee.Context) (err error) {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		workId := c.QueryParam("work_id")
		var msg string
		for {
			if workId != "" {
				var osc model.CoreSqlOrder
				model.DB().Model(model.CoreSqlOrder{}).Where("work_id =?", workId).Find(&osc)
				err := websocket.Message.Send(ws, osc.OSCInfo)
				if err != nil {
					c.Logger().Error(err)
					break
				}
			}
			if err := websocket.Message.Receive(ws, &msg); err != nil {
				break
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func AuditOrderApis(c yee.Context) (err error) {
	switch c.Params("tp") {
	case "state":
		return AuditOrderState(c)
	case "kill":
		return DelayKill(c)
	case "scheduled":
		return ScheduledChange(c)
	case "batch":
		return BatchAuditOrderState(c)
	case "batch_check":
		return BatchSQLCheckHandler(c)
	case "batch_rollback":
		return BatchRollbackHandler(c)
	default:
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_FAKE)))
	}
}

func BatchSQLCheckHandler(c yee.Context) (err error) {
	u := new(BatchConfirm)
	if err = c.Bind(u); err != nil {
		c.Logger().Error(err.Error())
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_BIND)))
	}
	if len(u.WorkIds) == 0 {
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE("work_ids is empty"))
	}
	results := BatchSQLCheck(u.WorkIds)
	return c.JSON(http.StatusOK, common.SuccessPayload(results))
}

func BatchRollbackHandler(c yee.Context) (err error) {
	u := new(BatchConfirm)
	user := new(factory.Token).JwtParse(c)
	if err = c.Bind(u); err != nil {
		c.Logger().Error(err.Error())
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_BIND)))
	}
	if len(u.WorkIds) == 0 {
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE("work_ids is empty"))
	}

	result := BatchResult{}

	for _, workId := range u.WorkIds {
		var order model.CoreSqlOrder
		if err := model.DB().Where("work_id = ?", workId).First(&order).Error; err != nil {
			result.Failed = append(result.Failed, BatchFailItem{WorkId: workId, Error: "工单不存在"})
			continue
		}
		if order.Status != 1 {
			result.Failed = append(result.Failed, BatchFailItem{WorkId: workId, Error: "工单状态不是已完成，无法回滚"})
			continue
		}

		var rolls []model.CoreRollback
		model.DB().Where("work_id = ?", workId).Find(&rolls)
		if len(rolls) == 0 {
			result.Failed = append(result.Failed, BatchFailItem{WorkId: workId, Error: "无回滚语句"})
			continue
		}

		var sqls []string
		for _, r := range rolls {
			sqls = append(sqls, r.SQL)
		}
		rollSQL := strings.Join(sqls, "\n")

		newOrder := model.CoreSqlOrder{
			SourceId: order.SourceId,
			SQL:      rollSQL,
			Type:     order.Type,
			IDC:      order.IDC,
			Source:   order.Source,
			DataBase: order.DataBase,
			Table:    order.Table,
			Delay:    "none",
			Backup:   0,
			Text:     "[批量回滚] " + order.Text,
		}

		var flowTpl model.CoreWorkflowTpl
		var flowSource model.CoreDataSource
		model.DB().Model(model.CoreDataSource{}).Where("source_id = ?", order.SourceId).First(&flowSource)
		model.DB().Model(model.CoreWorkflowTpl{}).Where("id = ?", flowSource.FlowID).First(&flowTpl)
		var steps []flow.Tpl
		if e := json.Unmarshal(flowTpl.Steps, &steps); e != nil || len(steps) < 2 {
			result.Failed = append(result.Failed, BatchFailItem{WorkId: workId, Error: "工作流配置异常"})
			continue
		}

		newOrder.WorkId = factory.GenWorkId()
		newOrder.Username = user.Username
		newOrder.RealName = user.RealName
		newOrder.Date = time.Now().Format("2006-01-02 15:04")
		newOrder.Status = 2
		newOrder.CurrentStep = 1
		newOrder.Assigned = strings.Join(steps[1].Auditor, ",")

		model.DB().Create(&newOrder)
		model.DB().Create(&model.CoreWorkflowDetail{
			WorkId:   newOrder.WorkId,
			Username: user.Username,
			Action:   "提交回滚工单",
			Time:     time.Now().Format("2006-01-02 15:04"),
		})
		pusher.NewMessagePusher(newOrder.WorkId).Order().OrderBuild(pusher.SummitStatus).Push()
		result.Success = append(result.Success, workId)
	}

	return c.JSON(http.StatusOK, common.SuccessPayload(result))
}

func AuditOrRecordOrderFetchApis(c yee.Context) (err error) {
	switch c.Params("tp") {
	//case "list":
	//	return FetchAuditOrder(c)
	//case "record":
	//	return FetchRecord(c)
	default:
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_FAKE)))
	}
}

func AuditFetchApis(c yee.Context) (err error) {
	switch c.Params("tp") {
	case "osc":
		return FetchOSCAPI(c)
	case "kill":
		return nil
	case "list":
		return FetchAuditOrder(c)
	default:
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_FAKE)))
	}
}
