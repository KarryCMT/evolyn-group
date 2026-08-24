-- 000017: RustFS 文件元数据。对象内容仅存 S3 兼容对象存储，files 保留租户
-- 归属、配额预留、访问授权和审计所需元数据；object_key 不经 API 出网。
CREATE TABLE IF NOT EXISTS files (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    code varchar(64) NOT NULL,
    bucket varchar(128) NOT NULL,
    object_key varchar(768) NOT NULL,
    original_name varchar(255) NOT NULL,
    content_type varchar(255) NOT NULL,
    declared_size BIGINT NOT NULL,
    actual_size BIGINT NOT NULL DEFAULT 0,
    sha256 varchar(64) NULL,
    state varchar(16) NOT NULL,
    expires_at timestamp with time zone NULL,
    creator_id BIGINT NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_files_state CHECK (state IN ('uploading', 'ready')),
    CONSTRAINT chk_files_declared_size CHECK (declared_size > 0),
    CONSTRAINT chk_files_actual_size CHECK (actual_size >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_files_tenant_code
    ON files (tenant_id, code)
    WHERE deleted_at IS NULL;

-- 存储配额统计与过期上传会话清理均按租户、状态过滤。
CREATE INDEX IF NOT EXISTS idx_files_tenant_state
    ON files (tenant_id, state)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_files_uploading_expiry
    ON files (expires_at)
    WHERE deleted_at IS NULL AND state = 'uploading';

COMMENT ON TABLE files IS '租户文件元数据：字节内容位于 RustFS；上传会话创建时占用 declared_size，确认后以 actual_size 计量';
COMMENT ON COLUMN files.code IS '对外文件标识，服务端生成 fil_ 前缀随机编码，租户内唯一';
COMMENT ON COLUMN files.object_key IS 'RustFS 私有对象键，仅服务端使用，不经 API 返回';
COMMENT ON COLUMN files.state IS 'uploading=已预留配额待直传确认；ready=服务端 StatObject 校验成功';
COMMENT ON COLUMN files.expires_at IS 'uploading 会话与上传预签名的失效时间，过期清理任务据此释放预留';

-- 存量租户的系统角色随资源一同补齐。资源目录由启动期 iamRepo.Init()
-- 幂等注册；规则 JSON 仅在尚未包含 files 时追加，避免重跑重复授权。
UPDATE roles
SET rules = (COALESCE(rules::jsonb, '[]'::jsonb) || jsonb_build_array(jsonb_build_object('resource', 'files', 'operation', '*')))::json
WHERE name = 'tenant-admin'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule::json->>'resource' = 'files'
  );

UPDATE roles
SET rules = (COALESCE(rules::jsonb, '[]'::jsonb) || jsonb_build_array(jsonb_build_object('resource', 'files', 'operation', 'edit')))::json
WHERE name = 'authenticated'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule::json->>'resource' = 'files'
  );
