package query

import (
	"github.com/cookieY/yee"
)

func AuditQueryRestFulAPis() yee.RestfulAPI {
	return yee.RestfulAPI{
		Get:    AuditQueryOrderApis,
		Put:    AuditQueryOrderProfileFetchApis,
		Delete: QueryDeleteEmptyRecord,
		Post:   QueryHandlerSets,
	}
}
