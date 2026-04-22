// Copyright 2019 HenryYee.
//
// Licensed under the AGPL, Version 3.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package enc

import (
	"Yearning-go/src/i18n"
	//"Yearning-go/src/model"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"github.com/cookieY/yee/logger"
)

func Encrypt(s, p string) string {

	if len(s) == 16 {
		// 转成字节数组
		origData := []byte(p)
		k := []byte(s)
		// 分组秘钥
		block, _ := aes.NewCipher(k)
		// 获取秘钥块的长度
		blockSize := block.BlockSize()
		// 补全码
		origData = PKCS7Padding(origData, blockSize)
		// 加密模式
		blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
		// 创建数组
		cryted := make([]byte, len(origData))
		// 加密
		blockMode.CryptBlocks(cryted, origData)

		return base64.StdEncoding.EncodeToString(cryted)
	}
	return ""
}

func Decrypt(s, cryted string) string {
	// 转成字节数组
	crytedByte, _ := base64.StdEncoding.DecodeString(cryted)
	k := []byte(s)

	// 分组秘钥
	block, _ := aes.NewCipher(k)
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
	// 创建数组
	orig := make([]byte, len(crytedByte))
	// 解密
	if (len(orig) % blockMode.BlockSize()) != 0 {
		return ""
	}
	blockMode.CryptBlocks(orig, crytedByte)
	//// 去补全码
	orig = PKCS7UnPadding(orig)
	if orig == nil {
		logger.DefaultLogger.Error(i18n.DefaultLang.Load(i18n.ER_KEY_DECRYPTION_FAILED))
		return ""
	}
	return string(orig)
}

// 补码
func PKCS7Padding(ciphertext []byte, blocksize int) []byte {
	padding := blocksize - len(ciphertext)%blocksize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// 去码
func PKCS7UnPadding(origData []byte) []byte {
	if origData == nil {
		return nil
	}
	if len(origData) > 0 {
		length := len(origData)
		unpadding := int(origData[length-1])
		if (length - unpadding) < 0 {
			return nil
		}
		return origData[:(length - unpadding)]
	}
	return nil
}
