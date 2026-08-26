package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseEnvironmentConfigTemplates 确保纳入版本管理的环境配置模板可被解析。
// app.local.yaml 为开发者私有配置且已忽略，不应由仓库测试依赖。
func TestParseEnvironmentConfigTemplates(t *testing.T) {
	for _, name := range []string{"app.test.yaml", "app.prod.yaml"} {
		t.Run(name, func(t *testing.T) {
			configPath := filepath.Join("..", "..", "config", name)
			_, err := Parse(configPath)
			require.NoError(t, err)
		})
	}
}

// TestParseLoadsPKIPrivateKeyFile 验证相对路径以配置文件目录为基准，避免依赖进程工作目录。
func TestParseLoadsPKIPrivateKeyFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "private_key.pem")
	configPath := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(keyPath, []byte("test private key"), 0o600))
	require.NoError(t, os.WriteFile(configPath, []byte("pki:\n  privateKeyFile: private_key.pem\n"), 0o600))

	conf, err := Parse(configPath)
	require.NoError(t, err)
	assert.Equal(t, "test private key", conf.PKI.PrivateKey)
}

// TestParseRejectsMultiplePKISources 确保私钥来源唯一，避免配置更新时误用旧密钥。
func TestParseRejectsMultiplePKISources(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "app.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("pki:\n  privateKey: inline\n  privateKeyFile: private_key.pem\n"), 0o600))

	_, err := Parse(configPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot both be configured")
}

// TestParseNormalizesAllowedOrigins CORS 白名单归一化（上线前复查 P2）：
// 空串/空白项在解析时丢弃——形如 [""] 的配置等价于未配置，
// release 的空白名单 fail-fast 不会被空项绕过
func TestParseNormalizesAllowedOrigins(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "app.yaml")
	require.NoError(t, os.WriteFile(configPath,
		[]byte("server:\n  allowedOrigins: [\"\", \"  \", \" https://a.com \", \"http://b.com\"]\n"), 0o600))

	conf, err := Parse(configPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://a.com", "http://b.com"}, conf.Server.AllowedOrigins)

	// 全空项归一化后为空列表
	require.NoError(t, os.WriteFile(configPath,
		[]byte("server:\n  allowedOrigins: [\"\", \"  \"]\n"), 0o600))
	conf, err = Parse(configPath)
	require.NoError(t, err)
	assert.Empty(t, conf.Server.AllowedOrigins)
}

// TestParseTreatsPlaceholderSecretAsUnconfigured 占位密钥视为未配置：
// 示例模板的 CHANGE_ME 被原样复制到生产时，若按「已配置」处理会绕过
// release 未配置告警，而 HMAC 密钥实际公开可预测（复查加固项）
func TestParseTreatsPlaceholderSecretAsUnconfigured(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "app.yaml")

	for _, placeholder := range []string{"CHANGE_ME", " change_me ", "Change-Me-KEEPED"} {
		// Change-Me-KEEPED 不是约定占位值，应原样保留（对照组）
		expectKept := placeholder != "CHANGE_ME" && placeholder != " change_me "
		require.NoError(t, os.WriteFile(configPath,
			[]byte("auth:\n  loginGuardSecret: \""+placeholder+"\"\n"), 0o600))

		conf, err := Parse(configPath)
		require.NoError(t, err)
		if expectKept {
			assert.Equal(t, placeholder, conf.Auth.LoginGuardSecret)
		} else {
			assert.Empty(t, conf.Auth.LoginGuardSecret, "placeholder %q should normalize to empty", placeholder)
		}
	}

	// 真实密钥原样保留
	require.NoError(t, os.WriteFile(configPath,
		[]byte("auth:\n  loginGuardSecret: \"real-secret\"\n"), 0o600))
	conf, err := Parse(configPath)
	require.NoError(t, err)
	assert.Equal(t, "real-secret", conf.Auth.LoginGuardSecret)
}
