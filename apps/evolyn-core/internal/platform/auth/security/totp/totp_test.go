package totp

import (
	"encoding/base32"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RFC 6238 附录 B 测试向量（SHA1，8 位值的 6 位截断）：
// 密钥为 ASCII "12345678901234567890"，base32 在测试内编码生成防手误
var rfcSecret = func() string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).
		EncodeToString([]byte("12345678901234567890"))
}()

func TestCodeRFCVectors(t *testing.T) {
	vectors := []struct {
		t    int64
		code string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
	}
	for _, v := range vectors {
		code, err := Code(rfcSecret, At(v.t))
		require.NoError(t, err)
		assert.Equal(t, v.code, code, "T=%d", v.t)
	}
}

func TestVerifyWindowAndReplay(t *testing.T) {
	at := At(1111111109)

	// 命中当前窗口
	code, _ := Code(rfcSecret, at)
	curCounter, ok, err := Verify(rfcSecret, code, at, 0)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, int64(1111111109/Step), curCounter)

	// 上一步窗口（drift -1 命中）：返回上一步计数器
	prevCode, _ := Code(rfcSecret, at-Step)
	prevCounter, ok, err := Verify(rfcSecret, prevCode, at, 0)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, int64(1111111109/Step)-1, prevCounter)

	// 重放保护：lastUsed 覆盖命中窗口后，同码与更早窗口均拒绝
	_, ok, err = Verify(rfcSecret, code, at, curCounter)
	require.NoError(t, err)
	assert.False(t, ok, "同窗口码不得重复使用")
	_, ok, err = Verify(rfcSecret, prevCode, at, curCounter)
	require.NoError(t, err)
	assert.False(t, ok, "更早窗口码不得回退使用")

	// 偏移超窗（2 步前的码）拒绝
	oldCode, _ := Code(rfcSecret, at-2*Step)
	_, ok, err = Verify(rfcSecret, oldCode, at, 0)
	require.NoError(t, err)
	assert.False(t, ok)

	// 错码/长度非法拒绝
	_, ok, err = Verify(rfcSecret, "000000", at, 0)
	require.NoError(t, err)
	assert.False(t, ok)
	_, ok, err = Verify(rfcSecret, "12345", at, 0)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestCipherRoundtrip(t *testing.T) {
	c, err := NewCipher("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	require.NoError(t, err)

	secret, err := GenerateSecret()
	require.NoError(t, err)

	ciphertext, err := c.Encrypt(secret)
	require.NoError(t, err)
	assert.NotEqual(t, secret, ciphertext)

	plain, err := c.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, secret, plain)

	// 篡改密文解密失败
	_, err = c.Decrypt(ciphertext[:len(ciphertext)-2] + "00")
	assert.Error(t, err)

	// 主密钥长度非法
	_, err = NewCipher("abcd")
	assert.Error(t, err)
}

func TestKeyringSupportsKeyRotation(t *testing.T) {
	const (
		keyV1 = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
		keyV2 = "ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100"
	)
	oldRing, err := NewKeyring(1, map[int]string{1: keyV1})
	require.NoError(t, err)
	oldCiphertext, oldVersion, err := oldRing.Encrypt("OLDSECRET")
	require.NoError(t, err)
	assert.Equal(t, 1, oldVersion)

	rotated, err := NewKeyring(2, map[int]string{1: keyV1, 2: keyV2})
	require.NoError(t, err)
	plain, err := rotated.Decrypt(oldVersion, oldCiphertext)
	require.NoError(t, err)
	assert.Equal(t, "OLDSECRET", plain)

	_, version, err := rotated.Encrypt("NEWSECRET")
	require.NoError(t, err)
	assert.Equal(t, 2, version)
	_, err = rotated.Decrypt(3, oldCiphertext)
	assert.Error(t, err)
}
