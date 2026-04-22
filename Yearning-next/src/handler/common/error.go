package common

// SOAR 错误码 1900-1999

func ERR_SOAR_ALTER_MERGE() Resp {
	return Resp{
		Code: 1901,
		Text: "sql is empty",
	}
}

func ERR_COMMON_MESSAGE(err error) Resp {
	return Resp{
		Code: 5555,
		Text: err.Error(),
	}
}

func ERR_COMMON_TEXT_MESSAGE(err string) Resp {
	return Resp{
		Code: 5555,
		Text: err,
	}
}
