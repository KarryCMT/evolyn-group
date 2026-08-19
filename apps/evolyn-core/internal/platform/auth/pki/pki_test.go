package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// encryptForTest 模拟前端 jsencrypt：以公钥 PEM 做 PKCS#1 v1.5 加密 + base64 输出。
// 公钥按 openssl/jsencrypt 的通行约定以 X.509 SPKI 解析
func encryptForTest(t *testing.T, kp *Keypair, plain string) string {
	t.Helper()
	block, _ := pem.Decode([]byte(kp.PublicKey))
	require.NotNil(t, block)
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	pub, ok := pubAny.(*rsa.PublicKey)
	require.True(t, ok)
	cipher, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(plain))
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(cipher)
}

func TestLoadGeneratesKeypairWhenEmpty(t *testing.T) {
	kp, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, "rsa", kp.Algorithm)
	assert.Contains(t, kp.PublicKey, "-----BEGIN PUBLIC KEY-----")

	// 2048 位密文 = 256 字节 → base64 后 344 字符
	cipher := encryptForTest(t, kp, "secret123")
	assert.Len(t, cipher, 344)
	plain, err := kp.Decrypt(cipher)
	require.NoError(t, err)
	assert.Equal(t, "secret123", plain)
}

func TestLoadParsesConfiguredPEM(t *testing.T) {
	// 生成一把 PKCS#1 私钥模拟配置注入，Load 应能解析并复用同一密钥
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	configured := string(pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))

	kp, err := Load(configured)
	require.NoError(t, err)

	cipher := encryptForTest(t, kp, "config-key")
	plain, err := kp.Decrypt(cipher)
	require.NoError(t, err)
	assert.Equal(t, "config-key", plain)

	// 密钥不匹配的密文（另一把密钥加密）解密必须失败
	other, err := Load("")
	require.NoError(t, err)
	_, err = kp.Decrypt(encryptForTest(t, other, "mismatch"))
	assert.ErrorIs(t, err, ErrDecrypt)
}

func TestDecryptRejectsMalformedInput(t *testing.T) {
	kp, err := Load("")
	require.NoError(t, err)

	// 非 base64
	_, err = kp.Decrypt("not-base64!!!")
	assert.ErrorIs(t, err, ErrDecrypt)
	// base64 但非合法密文
	_, err = kp.Decrypt(base64.StdEncoding.EncodeToString([]byte("junk")))
	assert.ErrorIs(t, err, ErrDecrypt)
}

func TestLoadRejectsBadPEM(t *testing.T) {
	_, err := Load("not a pem")
	assert.Error(t, err)
	_, err = Load("-----BEGIN RSA PRIVATE KEY-----\nbroken\n-----END RSA PRIVATE KEY-----")
	assert.Error(t, err)
}
