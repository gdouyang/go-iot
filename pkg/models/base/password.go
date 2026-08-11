package base

import (
	"crypto/md5"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost 登录/改密使用的 cost（≥10）。
const bcryptCost = 10

// HashPassword 使用 bcrypt 对明文密码哈希（新建用户 / 改密 / 懒迁移）。
func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", fmt.Errorf("password must be present")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// IsBcryptHash 判断存储串是否为 bcrypt（$2a$ / $2b$ / $2y$）。
func IsBcryptHash(stored string) bool {
	return strings.HasPrefix(stored, "$2a$") ||
		strings.HasPrefix(stored, "$2b$") ||
		strings.HasPrefix(stored, "$2y$")
}

// LegacyMD5Hash 旧版哈希：md5(username + password)，用于懒迁移校验。
func LegacyMD5Hash(username, plain string) string {
	sum := md5.Sum([]byte(username + plain))
	return fmt.Sprintf("%x", sum)
}

// CheckPassword 校验明文是否匹配库中哈希。
// 支持 bcrypt；不支持时回退 legacy MD5(username+password)。
// matched 表示密码正确；needUpgrade 表示应写回 bcrypt。
func CheckPassword(storedHash, username, plain string) (matched bool, needUpgrade bool) {
	if storedHash == "" || plain == "" {
		return false, false
	}
	if IsBcryptHash(storedHash) {
		err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(plain))
		return err == nil, false
	}
	// legacy MD5
	if storedHash == LegacyMD5Hash(username, plain) {
		return true, true
	}
	return false, false
}
