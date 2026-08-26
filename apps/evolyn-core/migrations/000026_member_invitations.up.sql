-- 成员邀请：邀请尚未被接受前不创建 users 记录，避免占用账号与成员配额。
CREATE TABLE IF NOT EXISTS member_invitations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    inviter_member_id BIGINT NOT NULL DEFAULT 0,
    name varchar(80) NOT NULL,
    identifier varchar(50),
    phone varchar(32),
    email varchar(256),
    profile jsonb NOT NULL DEFAULT '{}'::jsonb,
    invite_token varchar(64) NOT NULL,
    source varchar(16) NOT NULL DEFAULT 'manual',
    status varchar(16) NOT NULL DEFAULT 'pending',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_member_invitations_source CHECK (source IN ('manual', 'batch')),
    CONSTRAINT chk_member_invitations_status CHECK (status IN ('pending', 'accepted', 'cancelled')),
    CONSTRAINT chk_member_invitations_contact CHECK (phone <> '' OR email <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_member_invitations_token
    ON member_invitations (invite_token) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_member_invitations_tenant_phone_pending
    ON member_invitations (tenant_id, phone)
    WHERE phone <> '' AND status = 'pending' AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_member_invitations_tenant_email_pending
    ON member_invitations (tenant_id, email)
    WHERE email <> '' AND status = 'pending' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_member_invitations_tenant_status
    ON member_invitations (tenant_id, status, id DESC) WHERE deleted_at IS NULL;

-- 每个租户维护一条公开邀请链接；关闭后链接保留但不可使用，重新开启沿用同一安全 token。
CREATE TABLE IF NOT EXISTS tenant_public_invitation_links (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    token varchar(64) NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    creator_member_id BIGINT NOT NULL DEFAULT 0,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_public_invitation_links_tenant
    ON tenant_public_invitation_links (tenant_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_public_invitation_links_token
    ON tenant_public_invitation_links (token) WHERE deleted_at IS NULL;

COMMENT ON TABLE member_invitations IS '成员邀请记录：保存受邀人的完整档案草稿，接受邀请后再创建正式成员';
COMMENT ON COLUMN member_invitations.tenant_id IS '邀请所属租户 ID';
COMMENT ON COLUMN member_invitations.inviter_member_id IS '发起邀请的租户成员 ID';
COMMENT ON COLUMN member_invitations.name IS '受邀成员姓名，最长 80 字符';
COMMENT ON COLUMN member_invitations.identifier IS '企业内唯一成员编号，可由工号等承载';
COMMENT ON COLUMN member_invitations.phone IS '受邀手机号，须与邮箱至少填写一项';
COMMENT ON COLUMN member_invitations.email IS '受邀邮箱，须与手机号至少填写一项';
COMMENT ON COLUMN member_invitations.profile IS '邀请档案扩展字段：部门、别名、工号、性别、职务、聘用形式及日期等';
COMMENT ON COLUMN member_invitations.invite_token IS '单人邀请链接安全令牌';
COMMENT ON COLUMN member_invitations.source IS '邀请来源：manual 手动添加，batch 批量导入';
COMMENT ON COLUMN member_invitations.status IS '邀请状态：pending 待接受，accepted 已接受，cancelled 已取消';
COMMENT ON COLUMN member_invitations.created_at IS '邀请创建时间';
COMMENT ON COLUMN member_invitations.updated_at IS '邀请最后更新时间';
COMMENT ON COLUMN member_invitations.deleted_at IS '邀请软删除时间，NULL 表示有效';
COMMENT ON TABLE tenant_public_invitation_links IS '租户公开邀请链接开关与安全令牌';
COMMENT ON COLUMN tenant_public_invitation_links.tenant_id IS '链接所属租户 ID，每个租户仅一条有效记录';
COMMENT ON COLUMN tenant_public_invitation_links.token IS '公开邀请链接安全令牌';
COMMENT ON COLUMN tenant_public_invitation_links.enabled IS '是否允许通过公开链接加入租户';
COMMENT ON COLUMN tenant_public_invitation_links.creator_member_id IS '首次创建或最近更新链接的成员 ID';
COMMENT ON COLUMN tenant_public_invitation_links.created_at IS '公开链接创建时间';
COMMENT ON COLUMN tenant_public_invitation_links.updated_at IS '公开链接最后更新时间';
COMMENT ON COLUMN tenant_public_invitation_links.deleted_at IS '公开链接软删除时间，NULL 表示有效';
