package apis

import (
	"Yearning-go/src/handler/common"
	"Yearning-go/src/handler/fetch"
	"Yearning-go/src/handler/personal"
	"Yearning-go/src/i18n"
	"Yearning-go/src/lib/factory"
	"github.com/cookieY/yee"
	"net/http"
)

func YearningQueryForGet(y yee.Context) (err error) {
	tp := y.Params("tp")
	switch tp {
	case "tables":
		return personal.FetchQueryTableInfo(y)
	case "schema":
		return personal.FetchQueryDatabaseInfo(y)
	case "results":
		return personal.SocketQueryResults(y)
	}
	return y.JSON(http.StatusOK, "Illegal")
}

func YearningQueryForPut(y yee.Context) (err error) {
	tp := y.Params("tp")
	switch tp {
	case "merge":
		return fetch.FetchMergeDDL(y)
	}
	return y.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(i18n.DefaultLang.Load(i18n.ER_REQ_FAKE)))
}

func YearningQueryForPost(y yee.Context) (err error) {
	tp := y.Params("tp")
	user := new(factory.Token).JwtParse(y)
	switch tp {
	case "post":
		return personal.ReferQueryOrder(y, user)
	}
	return y.JSON(http.StatusOK, "Illegal")
}

func YearningQueryApis() yee.RestfulAPI {
	return yee.RestfulAPI{
		Get:    YearningQueryForGet,
		Put:    YearningQueryForPut,
		Post:   YearningQueryForPost,
		Delete: personal.UndoQueryOrder,
	}
}
