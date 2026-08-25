package totp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// Cipher TOTP 密钥的静态加密（AES-256-GCM）：密钥仅在 enroll 确认与
// 登录校验的内存中出现，落库恒为密文 + key_version（支持主密钥轮换：
// 换钥后新写入用新版本，旧因子按自身版本解密）
type Cipher struct {
	aead cipher.AEAD
}

// Keyring 管理 TOTP 主密钥轮换：新增因子固定使用 currentVersion，既有因子
// 根据数据库保存的 key_version 取回旧密钥解密。密钥仅驻留进程内存。
type Keyring struct {
	currentVersion int
	ciphers        map[int]*Cipher
}

// NewCipher 由主密钥构造（hex 编码的 32 字节 = AES-256）；主密钥由
// 生产环境配置注入，绝不入库/出网
func NewCipher(masterKeyHex string) (*Cipher, error) {
	key, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode totp master key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("totp master key must be 32 bytes (AES-256), got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// NewKeyring 从部署配置加载所有可用主密钥。调用方应在 release 环境启动时
// fail-fast，避免配置缺失导致 MFA 已开启账号无法登录。
func NewKeyring(currentVersion int, masterKeys map[int]string) (*Keyring, error) {
	if currentVersion <= 0 {
		return nil, fmt.Errorf("totp current key version must be positive")
	}
	if len(masterKeys) == 0 {
		return nil, fmt.Errorf("totp master keys are empty")
	}

	ciphers := make(map[int]*Cipher, len(masterKeys))
	for version, key := range masterKeys {
		if version <= 0 {
			return nil, fmt.Errorf("totp key version must be positive")
		}
		cipher, err := NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("load totp key version %d: %w", version, err)
		}
		ciphers[version] = cipher
	}
	if _, ok := ciphers[currentVersion]; !ok {
		return nil, fmt.Errorf("totp current key version %d is not configured", currentVersion)
	}
	return &Keyring{currentVersion: currentVersion, ciphers: ciphers}, nil
}

// Encrypt 使用当前版本加密，并把实际使用的版本一并交由调用方落库。
func (k *Keyring) Encrypt(plaintext string) (ciphertext string, keyVersion int, err error) {
	ciphertext, err = k.ciphers[k.currentVersion].Encrypt(plaintext)
	return ciphertext, k.currentVersion, err
}

// Decrypt 使用因子落库的版本对应密钥解密；缺少旧密钥应显式失败，而非静默
// 尝试当前密钥，以便部署方安全地完成轮换。
func (k *Keyring) Decrypt(keyVersion int, ciphertext string) (string, error) {
	cipher, ok := k.ciphers[keyVersion]
	if !ok {
		return "", fmt.Errorf("totp key version %d is not configured", keyVersion)
	}
	return cipher.Decrypt(ciphertext)
}

// Encrypt 密文格式：nonce(12) || ciphertext，hex 编码存储
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return hex.EncodeToString(c.aead.Seal(nonce, nonce, []byte(plaintext), nil)), nil
}

func (c *Cipher) Decrypt(ciphertext string) (string, error) {
	raw, err := hex.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode totp ciphertext: %w", err)
	}
	if len(raw) < c.aead.NonceSize() {
		return "", fmt.Errorf("totp ciphertext too short")
	}
	plain, err := c.aead.Open(nil, raw[:c.aead.NonceSize()], raw[c.aead.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt totp secret: %w", err)
	}
	return string(plain), nil
}
