package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
