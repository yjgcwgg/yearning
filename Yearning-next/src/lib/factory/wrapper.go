package factory

import (
	"Yearning-go/src/model"
	"crypto/sha256"
	"encoding/base64"
	"github.com/Jeffail/gabs/v2"
	"golang.org/x/crypto/pbkdf2"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

func ArrayRemove(source []byte, flag string) ([]byte, error) {
	p, err := gabs.ParseJSON(source)
	if err != nil {
		return nil, err
	}
	for i, c := range p.Children() {
		if c.Data().(string) == flag {
			_ = p.ArrayRemove(i)
		}
	}
	return p.EncodeJSON(), nil
}

func MultiArrayRemove(source []byte, sep []string, flag string) ([]byte, error) {
	// 解析JSON源
	p, err := gabs.ParseJSON(source)
	if err != nil {
		return nil, err // 返回解析错误
	}

	var wait sync.WaitGroup
	wait.Add(len(sep)) // 设置等待组

	for _, dl := range sep {
		go func(dl string) {
			defer wait.Done() // 确保在函数结束时调用Done
			children := p.S(dl).Children()

			// 通过切片收集待删除的索引
			var indicesToRemove []int
			for i, c := range children {
				if c.Data().(string) == flag {
					indicesToRemove = append(indicesToRemove, i)
				}
			}

			// 从后向前遍历以安全删除元素
			for i := len(indicesToRemove) - 1; i >= 0; i-- {
				p.ArrayRemove(indicesToRemove[i], dl)
			}
		}(dl) // 直接传递dl，避免闭包问题
	}

	wait.Wait()                // 等待所有goroutine完成
	return p.EncodeJSON(), nil // 返回更新后的JSON
}

func GetRandom() []byte {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	destr := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 12; i++ {
		result = append(result, destr[r.Intn(len(destr))])
	}
	return result
}

func DjangoEncrypt(password string, sl string) string {
	pwd := []byte(password)
	salt := []byte(sl)
	iterations := 120000
	digest := sha256.New
	dk := pbkdf2.Key(pwd, salt, iterations, 32, digest)
	str := base64.StdEncoding.EncodeToString(dk)
	return "pbkdf2_sha256" + "$" + strconv.FormatInt(int64(iterations), 10) + "$" + string(salt) + "$" + str
}

func DjangoCheckPassword(account *model.CoreAccount, password string) bool {
	sl := strings.Split(account.Password, "$")[2]
	checkPasswordToken := DjangoEncrypt(password, sl)
	if account.Password == checkPasswordToken {
		return true
	} else {
		return false
	}
}
