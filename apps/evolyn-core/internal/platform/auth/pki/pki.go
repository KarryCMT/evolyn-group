// Package pki 登录口令加密传输（认证域子能力）：RSA 公钥经 GET /app/conf
// 下发前端（pki 段），前端以公钥加密密码明文上送，服务端持私钥解密后再
// 走 bcrypt 校验——避免明文口令经过传输层。填充方式 PKCS#1 v1.5，
// 与前端 jsencrypt 默认行为对齐
package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"

	"evolyn/internal/platform/httpx"
)

// Keypair RSA 密钥对：Algorithm 固定 rsa；PublicKey 为 PEM（下发前端），
// 私钥仅留在进程内做解密
type Keypair struct {
	Algorithm  string
	PublicKey  string
	privateKey *rsa.PrivateKey
}

// keyBits 私钥位数：2048 为当前通行下限（简道云示例为弱 512 位，不采用）
const keyBits = 2048

// Load 加载密钥对：privateKeyPEM 非空时解析配置私钥（生产路径，多实例
// 必须配置同一密钥对）；为空时启动随机生成（仅开发/测试——重启即轮换，
// 前端每次页面加载重新拉取公钥所以不影响联调，但多实例会解密互斥）
func Load(privateKeyPEM string) (*Keypair, error) {
	if privateKeyPEM == "" {
		key, err := rsa.GenerateKey(rand.Reader, keyBits)
		if err != nil {
			return nil, fmt.Errorf("generate rsa key: %w", err)
		}
		return newKeypair(key), nil
	}

	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("pki private key is not valid PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// 兼容 PKCS#8 格式导出的私钥
		key8, err8 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err8 != nil {
			return nil, fmt.Errorf("parse pki private key: %w", err)
		}
		key, ok := key8.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("pki private key is not an RSA key")
		}
		return newKeypair(key), nil
	}
	return newKeypair(key), nil
}

func newKeypair(key *rsa.PrivateKey) *Keypair {
	// 公钥按 X.509 SubjectPublicKeyInfo 编码：这是 "BEGIN PUBLIC KEY" PEM
	// 的通行约定（openssl/jsencrypt 均按 SPKI 解析；PKCS#1 裸结构会解析失败）
	spki, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		// rsa.PublicKey 转 SPKI 仅在算法不支持时失败，RSA 恒可编码
		panic(fmt.Sprintf("marshal rsa public key: %v", err))
	}
	return &Keypair{
		Algorithm:  "rsa",
		PublicKey:  string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: spki})),
		privateKey: key,
	}
}

// Decrypt 解密前端上送的口令密文（jsencrypt 输出为 base64 + PKCS#1 v1.5）；
// 任何失败均返回同一错误文案，不区分「未加密/密文损坏/密钥不匹配」，
// 避免向调用方泄露判定细节
func (k *Keypair) Decrypt(cipherBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(cipherBase64)
	if err != nil {
		return "", ErrDecrypt
	}
	plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, k.privateKey, ciphertext)
	if err != nil {
		return "", ErrDecrypt
	}
	return string(plaintext), nil
}

// ErrDecrypt 口令解密失败（统一语义：前端需加密上送并与 /app/conf 公钥匹配）。
// ADR-008 稳定码：密文细节不出网，前端可提示刷新页面重取公钥
var ErrDecrypt = httpx.NewBiz("AUTH_PASSWORD_DECRYPT_FAILED", "密码传输校验失败，请刷新页面后重试", http.StatusBadRequest)
