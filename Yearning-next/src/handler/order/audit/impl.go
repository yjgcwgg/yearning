package audit

import (
	"Yearning-go/src/engine"
	"Yearning-go/src/handler/common"
	"Yearning-go/src/handler/manage/flow"
	"Yearning-go/src/i18n"
	"Yearning-go/src/lib/calls"
	"Yearning-go/src/lib/enc"
	"Yearning-go/src/lib/factory"
	"Yearning-go/src/lib/pusher"
	"Yearning-go/src/lib/vars"
	"Yearning-go/src/model"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cookieY/yee/logger"
)

type ExecArgs struct {
	Order         *model.CoreSqlOrder
	Rules         engine.AuditRole
	IP            string
	Port          int
	Username      string
	Password      string
	CA            string
	Cert          string
	Key           string
	Message       model.Message
	MaxAffectRows uint
}

type Confirm struct {
	WorkId   string `json:"work_id"`
	Page     int    `json:"page"`
	Flag     int    `json:"flag"`
	Text     string `json:"text"`
	Tp       string `json:"tp"`
	SourceId string `json:"source_id"`
	Delay    string `json:"delay"`
}

type BatchConfirm struct {
	WorkIds []string `json:"work_ids"`
	Tp      string   `json:"tp"`
	Text    string   `json:"text"`
}

type BatchResult struct {
	Success []string        `json:"success"`
	Failed  []BatchFailItem `json:"failed"`
}

type BatchFailItem struct {
	WorkId string `json:"work_id"`
	Error  string `json:"error"`
}

func (e *Confirm) GetTPL() []flow.Tpl {
	var s model.CoreDataSource
	var tpl []flow.Tpl
	var flow model.CoreWorkflowTpl
	model.DB().Model(model.CoreDataSource{}).Select("flow_id").Where("source_id =?", e.SourceId).First(&s)
	model.DB().Model(model.CoreWorkflowTpl{}).Where("id =?", s.FlowID).First(&flow)
	_ = json.Unmarshal(flow.Steps, &tpl)
	return tpl
}

func ExecuteOrder(u *Confirm, user string) common.Resp {
	var order model.CoreSqlOrder
	var source model.CoreDataSource
	model.DB().Where("work_id =?", u.WorkId).First(&order)

	if order.Status != 2 && order.Status != 5 {
		return common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ORDER_NOT_SEARCH))
	}
	order.Assigned = user

	model.DB().Model(model.CoreDataSource{}).Where("source_id =?", order.SourceId).First(&source)
	rule, err := factory.CheckDataSourceRule(source.RuleId)
	if err != nil {
		logger.DefaultLogger.Error(err)
	}

	origBackup := order.Backup
	if order.Type == vars.DML && order.Backup == 1 {
		order.Backup = 0
	} else if origBackup == 1 {
		rule.PRIRollBack = true
	}

	var isCall bool
	if client := calls.NewRpc(); client != nil {
		if err := client.Call("Engine.Exec", &ExecArgs{
			Order:         &order,
			Rules:         *rule,
			IP:            source.IP,
			Port:          source.Port,
			Username:      source.Username,
			Password:      enc.Decrypt(model.C.General.SecretKey, source.Password),
			CA:            source.CAFile,
			Cert:          source.Cert,
			Key:           source.KeyFile,
			Message:       model.GloMessage,
			MaxAffectRows: rule.MaxAffectRows,
		}, &isCall); err != nil {
			return common.ERR_COMMON_MESSAGE(err)
		}

		if order.Type == vars.DML && origBackup == 1 {
			go GenerateDMLRollback(&source, u.WorkId, order.DataBase, order.SQL)
		}

		model.DB().Create(&model.CoreWorkflowDetail{
			WorkId:   u.WorkId,
			Username: user,
			Time:     time.Now().Format("2006-01-02 15:04"),
			Action:   i18n.DefaultLang.Load(i18n.ORDER_EXECUTE_STATE),
		})
		return common.SuccessPayLoadToMessage(i18n.DefaultLang.Load(i18n.ORDER_EXECUTE_STATE))
	}
	return common.ERR_COMMON_MESSAGE(fmt.Errorf("SQL引擎(Juno)未启动，无法执行工单。请确认引擎已运行在 %s", model.C.General.RpcAddr))

}

func GenerateDMLRollback(source *model.CoreDataSource, workId, schema, sqlText string) {
	db, err := source.ConnectDB(schema)
	if err != nil {
		logger.DefaultLogger.Error(fmt.Sprintf("rollback: connect target db failed: %v", err))
		return
	}
	defer model.Close(db)

	stmts := splitStatements(sqlText)
	for _, stmt := range stmts {
		rollSQL := generateReverseSQL(db, schema, stmt)
		if rollSQL != "" {
			model.DB().Create(&model.CoreRollback{WorkId: workId, SQL: rollSQL})
		}
	}
}

func splitStatements(sql string) []string {
	var stmts []string
	for _, s := range strings.Split(sql, ";") {
		s = strings.TrimSpace(s)
		if s != "" {
			stmts = append(stmts, s)
		}
	}
	return stmts
}

var reInsert = regexp.MustCompile(`(?i)^\s*INSERT\s+INTO\s+` + "`?" + `(\w+)` + "`?" + `\s*\(([^)]+)\)\s*(?:VALUE|VALUES)\s*\(([^)]+)\)`)

func generateReverseSQL(db interface{}, schema, stmt string) string {
	upper := strings.ToUpper(strings.TrimSpace(stmt))
	if strings.HasPrefix(upper, "INSERT") {
		return reverseInsert(schema, stmt)
	}
	if strings.HasPrefix(upper, "UPDATE") || strings.HasPrefix(upper, "DELETE") {
		return fmt.Sprintf("-- [需手动回滚] 原语句: %s", stmt)
	}
	return ""
}

func reverseInsert(schema, stmt string) string {
	m := reInsert.FindStringSubmatch(stmt)
	if m == nil {
		return fmt.Sprintf("-- [需手动回滚] 原语句: %s", stmt)
	}
	table := m[1]
	cols := strings.Split(m[2], ",")
	vals := strings.Split(m[3], ",")

	if len(cols) != len(vals) {
		return fmt.Sprintf("-- [需手动回滚] 原语句: %s", stmt)
	}

	var where []string
	for i, col := range cols {
		col = strings.TrimSpace(col)
		col = strings.Trim(col, "`")
		val := strings.TrimSpace(vals[i])
		where = append(where, fmt.Sprintf("`%s` = %s", col, val))
	}

	if schema != "" {
		return fmt.Sprintf("DELETE FROM `%s`.`%s` WHERE %s;", schema, table, strings.Join(where, " AND "))
	}
	return fmt.Sprintf("DELETE FROM `%s` WHERE %s;", table, strings.Join(where, " AND "))
}

func MultiAuditOrder(req *Confirm, user string) common.Resp {
	if assigned, isExecute, ok := isNotIdempotent(req, user); ok {
		if isExecute {
			return ExecuteOrder(req, user)
		}
		model.DB().Model(model.CoreSqlOrder{}).Where("work_id = ?", req.WorkId).Updates(&model.CoreSqlOrder{CurrentStep: req.Flag + 1, Assigned: strings.Join(assigned, ",")})
		model.DB().Create(&model.CoreWorkflowDetail{
			WorkId:   req.WorkId,
			Username: user,
			Time:     time.Now().Format("2006-01-02 15:04"),
			Action:   fmt.Sprintf(i18n.DefaultLang.Load(i18n.ORDER_AGREE_MESSAGE), strings.Join(assigned, " ")),
		})
		pusher.NewMessagePusher(req.WorkId).Order().OrderBuild(pusher.NextStepStatus).Push()
		return common.SuccessPayLoadToMessage(i18n.DefaultLang.Load(i18n.ORDER_AGREE_STATE))
	}
	return common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ORDER_NOT_SEARCH))
}

func RejectOrder(req *Confirm, user string) common.Resp {
	model.DB().Model(&model.CoreSqlOrder{}).Where("work_id =?", req.WorkId).Updates(map[string]interface{}{"status": 0})
	model.DB().Create(&model.CoreWorkflowDetail{
		WorkId:   req.WorkId,
		Username: user,
		Time:     time.Now().Format("2006-01-02 15:04"),
		Action:   i18n.DefaultLang.Load(i18n.ORDER_REJECT_MESSAGE),
	})
	model.DB().Create(&model.CoreOrderComment{
		WorkId:   req.WorkId,
		Username: user,
		Content:  fmt.Sprintf("驳回理由: %s", req.Text),
		Time:     time.Now().Format("2006-01-02 15:04"),
	})
	pusher.NewMessagePusher(req.WorkId).Order().OrderBuild(pusher.RejectStatus).Push()
	return common.SuccessPayLoadToMessage(i18n.DefaultLang.Load(i18n.ORDER_REJECT_STATE))
}

type BatchCheckItem struct {
	WorkId  string          `json:"work_id"`
	Source  string          `json:"source"`
	SQL     string          `json:"sql"`
	Results []engine.Record `json:"results"`
	Error   string          `json:"error"`
}

func BatchSQLCheck(workIds []string) []BatchCheckItem {
	var items []BatchCheckItem
	client := calls.NewRpc()

	for _, workId := range workIds {
		var order model.CoreSqlOrder
		if err := model.DB().Where("work_id = ?", workId).First(&order).Error; err != nil {
			items = append(items, BatchCheckItem{WorkId: workId, Error: "工单不存在"})
			continue
		}

		var source model.CoreDataSource
		model.DB().Where("source_id = ?", order.SourceId).First(&source)
		rule, err := factory.CheckDataSourceRule(source.RuleId)
		if err != nil {
			items = append(items, BatchCheckItem{WorkId: workId, Source: source.Source, SQL: order.SQL, Error: err.Error()})
			continue
		}

		if client == nil {
			items = append(items, BatchCheckItem{
				WorkId: workId, Source: source.Source, SQL: order.SQL,
				Results: []engine.Record{{SQL: order.SQL, Level: 0, Status: "warn", Error: "SQL引擎未启动，跳过语法检测"}},
			})
			continue
		}

		var rs []engine.Record
		if err := client.Call("Engine.Check", engine.CheckArgs{
			SQL:      order.SQL,
			Schema:   order.DataBase,
			IP:       source.IP,
			Username: source.Username,
			Port:     source.Port,
			Password: enc.Decrypt(model.C.General.SecretKey, source.Password),
			CA:       source.CAFile,
			Cert:     source.Cert,
			Key:      source.KeyFile,
			Kind:     order.Type,
			Lang:     model.C.General.Lang,
			Rule:     *rule,
		}, &rs); err != nil {
			items = append(items, BatchCheckItem{WorkId: workId, Source: source.Source, SQL: order.SQL, Error: err.Error()})
			continue
		}

		items = append(items, BatchCheckItem{WorkId: workId, Source: source.Source, SQL: order.SQL, Results: rs})
	}
	return items
}

func delayKill(workId string) string {
	model.DB().Model(&model.CoreSqlOrder{}).Where("work_id =?", workId).Updates(map[string]interface{}{"status": 4, "execute_time": time.Now().Format("2006-01-02 15:04"), "is_kill": 1})
	return i18n.DefaultLang.Load(i18n.ORDER_DELAY_KILL_DETAIL)
}

func isNotIdempotent(r *Confirm, user string) ([]string, bool, bool) {
	tpl := r.GetTPL()
	if len(tpl) > r.Flag {
		pList := strings.Join(tpl[r.Flag].Auditor, ",")
		if !strings.Contains(pList, user) {
			return nil, false, false
		}
		if r.Flag+1 == len(tpl) {
			return tpl[r.Flag].Auditor, true, true
		}
		return tpl[r.Flag+1].Auditor, false, true
	}
	return nil, false, false
}
