package personal

import (
	"Yearning-go/src/handler/common"
	"Yearning-go/src/lib/factory"
	"Yearning-go/src/model"
	"bytes"
	"encoding/base64"
	"github.com/cookieY/yee"
	"github.com/pquerna/otp/totp"
	"image/png"
	"net/http"
)

type mfaCodeReq struct {
	Code string `json:"code"`
}

func MFASetup(c yee.Context) error {
	user := new(factory.Token).JwtParse(c)

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Yearning",
		AccountName: user.Username,
	})
	if err != nil {
		return c.JSON(http.StatusOK, common.ERR_COMMON_MESSAGE(err))
	}

	model.DB().Model(&model.CoreAccount{}).
		Where("username = ?", user.Username).
		Update("mfa_secret", key.Secret())

	img, err := key.Image(200, 200)
	if err != nil {
		return c.JSON(http.StatusOK, common.ERR_COMMON_MESSAGE(err))
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return c.JSON(http.StatusOK, common.ERR_COMMON_MESSAGE(err))
	}
	qrCode := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	return c.JSON(http.StatusOK, common.SuccessPayload(map[string]interface{}{
		"secret":  key.Secret(),
		"qr_code": qrCode,
	}))
}

func MFAVerify(c yee.Context) error {
	user := new(factory.Token).JwtParse(c)
	u := new(mfaCodeReq)
	if err := c.Bind(u); err != nil {
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE("invalid request"))
	}

	var account model.CoreAccount
	model.DB().Where("username = ?", user.Username).First(&account)

	if account.MFASecret == "" {
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE("请先执行 MFA 设置"))
	}

	if !totp.Validate(u.Code, account.MFASecret) {
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE("验证码错误"))
	}

	model.DB().Model(&model.CoreAccount{}).
		Where("username = ?", user.Username).
		Update("mfa_enabled", true)

	return c.JSON(http.StatusOK, common.SuccessPayLoadToMessage("MFA 已启用"))
}

func MFADisable(c yee.Context) error {
	user := new(factory.Token).JwtParse(c)
	u := new(mfaCodeReq)
	if err := c.Bind(u); err != nil {
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE("invalid request"))
	}

	var account model.CoreAccount
	model.DB().Where("username = ?", user.Username).First(&account)

	if !account.MFAEnabled {
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE("MFA 未启用"))
	}

	if !totp.Validate(u.Code, account.MFASecret) {
		return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE("验证码错误"))
	}

	model.DB().Model(&model.CoreAccount{}).
		Where("username = ?", user.Username).
		Updates(map[string]interface{}{
			"mfa_enabled": false,
			"mfa_secret":  "",
		})

	return c.JSON(http.StatusOK, common.SuccessPayLoadToMessage("MFA 已关闭"))
}

func MFAStatus(c yee.Context) error {
	user := new(factory.Token).JwtParse(c)
	var account model.CoreAccount
	model.DB().Select("mfa_enabled").Where("username = ?", user.Username).First(&account)
	return c.JSON(http.StatusOK, common.SuccessPayload(map[string]interface{}{
		"mfa_enabled": account.MFAEnabled,
	}))
}
