-- 000039: 消息中心 P1+P2（docs/低代码平台/消息中心/消息中心现状分析与后端开发设计.md）。
-- 三类数据分表：不可变逻辑消息 notification_messages（多成员共享内容）+
-- 成员收件箱/已读 notification_member_inboxes + 租户通知设置聚合
--（tenant_notification_settings / preferences / preference_recipients /
-- custom_recipients）。业务域经事务 Outbox（notification_outbox_events）发布
-- 结构化事件，Worker 领取后渲染纯文本快照并扇出站内信。
-- 权限：notifications:view/update 授全体成员（authenticated），
-- notification-settings:* 按管理员规则签名补授租户管理员。

-- 不可变逻辑消息：一条消息多成员共享，不重复存储大段文本；无软删，
-- 过期后由保留清理 Worker 成批硬删（expires_at = occurred_at + 保留期）
CREATE TABLE IF NOT EXISTS notification_messages (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    event_id varchar(128) NOT NULL,
    category_code varchar(64) NOT NULL,
    event_code varchar(128) NOT NULL,
    severity varchar(16) NOT NULL DEFAULT 'info',
    title varchar(200) NOT NULL DEFAULT '',
    content varchar(2000) NOT NULL,
    action JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_ref JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone,
    CONSTRAINT chk_notification_messages_severity CHECK (severity IN ('info', 'success', 'warning', 'error')),
    CONSTRAINT chk_notification_messages_action CHECK (jsonb_typeof(action) = 'object'),
    CONSTRAINT chk_notification_messages_source_ref CHECK (jsonb_typeof(source_ref) = 'object'),
    CONSTRAINT chk_notification_messages_content CHECK (length(btrim(content)) > 0)
);

-- Worker 重试幂等：同 (tenant_id, event_id) 重复物化直接吞并
CREATE UNIQUE INDEX IF NOT EXISTS uk_notification_messages_tenant_event
    ON notification_messages (tenant_id, event_id);

-- 租户内按分类排查；常规成员查询从收件箱表进入
CREATE INDEX IF NOT EXISTS idx_notification_messages_tenant_category
    ON notification_messages (tenant_id, category_code, occurred_at DESC, id DESC);

-- 成员收件箱：站内信扇出与已读状态；category_code/occurred_at 为不可变冗余，
-- 支撑列表索引与稳定排序（occurred_at 相同以 id 决胜）
CREATE TABLE IF NOT EXISTS notification_member_inboxes (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    message_id BIGINT NOT NULL REFERENCES notification_messages(id),
    member_id BIGINT NOT NULL,
    category_code varchar(64) NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    read_at timestamp with time zone,
    created_at timestamp with time zone
);

-- 扇出去重：同消息同成员唯一收件箱行（Worker 重放幂等的第二道闸）
CREATE UNIQUE INDEX IF NOT EXISTS uk_notification_inboxes_unique
    ON notification_member_inboxes (tenant_id, message_id, member_id);

-- 成员分类列表（occurred_at DESC, id DESC 游标排序同构）
CREATE INDEX IF NOT EXISTS idx_notification_inboxes_member_list
    ON notification_member_inboxes (tenant_id, member_id, category_code, occurred_at DESC, id DESC);

-- 未读分类计数/列表
CREATE INDEX IF NOT EXISTS idx_notification_inboxes_member_unread
    ON notification_member_inboxes (tenant_id, member_id, category_code, occurred_at DESC, id DESC)
    WHERE read_at IS NULL;

-- 租户通知设置聚合根：每租户一行有效记录，revision 覆盖整个
-- 偏好/接收规则/自定义联系人聚合的乐观锁
CREATE TABLE IF NOT EXISTS tenant_notification_settings (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    revision BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_notification_settings_tenant
    ON tenant_notification_settings (tenant_id)
    WHERE deleted_at IS NULL;

-- 事件偏好覆盖：只保存对事件注册表默认值的覆盖，无覆盖行时投影注册表默认
CREATE TABLE IF NOT EXISTS tenant_notification_preferences (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    setting_id BIGINT NOT NULL REFERENCES tenant_notification_settings(id),
    event_code varchar(128) NOT NULL,
    system_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    email_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    sms_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    recipients_overridden BOOLEAN NOT NULL DEFAULT FALSE,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

-- 租户内事件码唯一（一个事件至多一条覆盖行）
CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_notification_prefs_event
    ON tenant_notification_preferences (tenant_id, event_code);

CREATE INDEX IF NOT EXISTS idx_tenant_notification_prefs_setting
    ON tenant_notification_preferences (setting_id);

-- 事件偏好的接收规则关联：动态规则（event_actor/event_audience/tenant_admin）
-- 同一偏好内至多一条，custom_recipient 不能重复关联（服务层校验）
CREATE TABLE IF NOT EXISTS tenant_notification_preference_recipients (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    preference_id BIGINT NOT NULL REFERENCES tenant_notification_preferences(id),
    target_kind varchar(32) NOT NULL,
    custom_recipient_id BIGINT,
    created_at timestamp with time zone,
    CONSTRAINT chk_tenant_notif_pref_recipients_kind CHECK (target_kind IN ('event_actor', 'event_audience', 'tenant_admin', 'custom_recipient')),
    CONSTRAINT chk_tenant_notif_pref_recipients_custom CHECK (
        (target_kind = 'custom_recipient' AND custom_recipient_id IS NOT NULL)
        OR (target_kind <> 'custom_recipient' AND custom_recipient_id IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_tenant_notif_pref_recipients_pref
    ON tenant_notification_preference_recipients (preference_id);

-- 自定义联系人删除前的引用检查（在用则拒绝删除）
CREATE INDEX IF NOT EXISTS idx_tenant_notif_pref_recipients_recipient
    ON tenant_notification_preference_recipients (custom_recipient_id)
    WHERE custom_recipient_id IS NOT NULL;

-- 租户级自定义外部提醒对象池：仅用于邮件/短信（无成员身份，不产生站内收件箱）；
-- 软删除保留关联历史；手机/邮箱至少一项由服务层校验
CREATE TABLE IF NOT EXISTS tenant_notification_custom_recipients (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    name varchar(80) NOT NULL,
    mobile varchar(32) NOT NULL DEFAULT '',
    email varchar(254) NOT NULL DEFAULT '',
    revision BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE INDEX IF NOT EXISTS idx_tenant_notification_recipients_tenant
    ON tenant_notification_custom_recipients (tenant_id)
    WHERE deleted_at IS NULL;

-- 事务 Outbox：业务事务与消息物化之间的可靠边界；Worker 以
-- FOR UPDATE SKIP LOCKED 小批领取，重试按 event_id 幂等
CREATE TABLE IF NOT EXISTS notification_outbox_events (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    event_id varchar(128) NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    event_code varchar(128) NOT NULL,
    actor_member_id BIGINT NOT NULL DEFAULT 0,
    audience_member_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP,
    status varchar(16) NOT NULL DEFAULT 'pending',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP,
    last_error_code varchar(100) NOT NULL DEFAULT '',
    created_at timestamp with time zone,
    processed_at timestamp with time zone,
    CONSTRAINT chk_notification_outbox_status CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    CONSTRAINT chk_notification_outbox_audience CHECK (jsonb_typeof(audience_member_ids) = 'array'),
    CONSTRAINT chk_notification_outbox_parameters CHECK (jsonb_typeof(parameters) = 'object')
);

-- 生产者事件唯一 ID 全局幂等
CREATE UNIQUE INDEX IF NOT EXISTS uk_notification_outbox_event_id
    ON notification_outbox_events (event_id);

-- Worker 领取扫描（status + 到期时间 + 顺序）
CREATE INDEX IF NOT EXISTS idx_notification_outbox_dispatch
    ON notification_outbox_events (status, next_attempt_at, id);

-- 消息中心（000039）：基线管理员按规则签名补授 notification-settings 资源
--（口径同 000035/000037，与可改名的角色名无关）
UPDATE roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'notification-settings', 'operation', '*'))
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*'
  )
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'notification-settings'
  );

-- 消息中心（000039）：全体成员读写自己的收件箱（view 覆盖摘要/列表，
-- update 覆盖单条/批量已读；数据范围由 Repository 的 tenant_id+member_id 双条件兜底）
UPDATE roles
SET rules = (
    rules::jsonb
    || '[{"resource": "notifications", "operation": "view"}, {"resource": "notifications", "operation": "update"}]'::jsonb
)::json
WHERE name = 'authenticated'
  AND deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'notifications'
  );

COMMENT ON TABLE notification_messages IS '消息中心不可变逻辑消息（000039）：模板渲染后的展示快照固化存储，多成员经收件箱共享同一行；action 仅登记稳定动作码与受控参数；按保留策略成批硬删，不是合规审计表';
COMMENT ON COLUMN notification_messages.id IS '自增主键（逻辑消息 ID，游标决胜值）';
COMMENT ON COLUMN notification_messages.tenant_id IS '归属租户 ID';
COMMENT ON COLUMN notification_messages.event_id IS '生产者事件唯一 ID，与租户组合唯一，承担 Worker 重试幂等';
COMMENT ON COLUMN notification_messages.category_code IS '稳定分类码（八个分类之一，只增不改）';
COMMENT ON COLUMN notification_messages.event_code IS '稳定事件码（事件注册表键，只增不改）';
COMMENT ON COLUMN notification_messages.severity IS '严重等级：info/success/warning/error';
COMMENT ON COLUMN notification_messages.title IS '可选短标题（纯文本，空串表示无标题）';
COMMENT ON COLUMN notification_messages.content IS '模板渲染后的展示快照（纯文本，非空，历史消息不随模板修改回写）';
COMMENT ON COLUMN notification_messages.action IS '受控跳转动作 JSONB：仅稳定动作码与白名单参数，默认空对象；前端按动作注册表映射路由';
COMMENT ON COLUMN notification_messages.source_ref IS '应用/表单/任务等可选追溯引用 JSONB，不出网';
COMMENT ON COLUMN notification_messages.occurred_at IS '业务事件发生时间（列表排序与游标第一因子）';
COMMENT ON COLUMN notification_messages.expires_at IS '读取与未读统计的有效期上界（occurred_at + 保留期，默认六个月）；过期即从列表/摘要排除';
COMMENT ON COLUMN notification_messages.created_at IS '消息物化时间（Worker 扇出落库时间）';
COMMENT ON TABLE notification_member_inboxes IS '消息中心成员收件箱（000039）：站内信扇出与已读状态；所有查询/更新必须显式携带 tenant_id + member_id 双条件';
COMMENT ON COLUMN notification_member_inboxes.id IS '自增主键（游标后备决胜值）';
COMMENT ON COLUMN notification_member_inboxes.tenant_id IS '冗余租户条件（所有查询必填）';
COMMENT ON COLUMN notification_member_inboxes.message_id IS '逻辑消息 ID（外键指向 notification_messages）';
COMMENT ON COLUMN notification_member_inboxes.member_id IS '接收成员 ID（租户内成员）';
COMMENT ON COLUMN notification_member_inboxes.category_code IS '不可变分类冗余（扇出时固化），用于分类列表/未读索引免 JOIN';
COMMENT ON COLUMN notification_member_inboxes.occurred_at IS '不可变事件时间冗余（扇出时固化），用于稳定排序';
COMMENT ON COLUMN notification_member_inboxes.read_at IS '已读时间；NULL=未读，重复标记已读不改写首次时间';
COMMENT ON COLUMN notification_member_inboxes.created_at IS '扇出时间';
COMMENT ON TABLE tenant_notification_settings IS '租户通知设置聚合根（000039）：每租户一行有效记录，租户开通事务预置、读取侧幂等兜底；revision 覆盖偏好/接收规则/自定义联系人整个聚合';
COMMENT ON COLUMN tenant_notification_settings.id IS '自增主键';
COMMENT ON COLUMN tenant_notification_settings.tenant_id IS '归属租户 ID（有效记录唯一）';
COMMENT ON COLUMN tenant_notification_settings.revision IS '聚合乐观锁口令：任何偏好/接收规则/联系人变更使其 +1，客户端原样回传，过期返回 409';
COMMENT ON COLUMN tenant_notification_settings.created_at IS '创建时间';
COMMENT ON COLUMN tenant_notification_settings.updated_at IS '更新时间';
COMMENT ON COLUMN tenant_notification_settings.deleted_at IS '软删除时间，NULL=未删除';
COMMENT ON TABLE tenant_notification_preferences IS '租户事件通知偏好覆盖（000039）：只保存对事件注册表默认值的覆盖，无覆盖行时投影注册表默认值；新事件入注册表无需为存量租户回填';
COMMENT ON COLUMN tenant_notification_preferences.id IS '自增主键';
COMMENT ON COLUMN tenant_notification_preferences.tenant_id IS '归属租户 ID（与 setting_id 冗余一致，服务层写入前复核）';
COMMENT ON COLUMN tenant_notification_preferences.setting_id IS '所属设置聚合根 ID';
COMMENT ON COLUMN tenant_notification_preferences.event_code IS '事件注册表事件码（租户内唯一）';
COMMENT ON COLUMN tenant_notification_preferences.system_enabled IS '站内信开关；受注册表必选渠道限制（system 为必选时不可关闭）';
COMMENT ON COLUMN tenant_notification_preferences.email_enabled IS '邮件渠道开关（能力未就绪时服务端拒绝新开启）';
COMMENT ON COLUMN tenant_notification_preferences.sms_enabled IS '短信渠道开关（能力未就绪时服务端拒绝新开启，不保存无法发送的虚假状态）';
COMMENT ON COLUMN tenant_notification_preferences.recipients_overridden IS '区分「使用注册表默认接收对象」（false）与「显式配置接收规则」（true，含显式清空）';
COMMENT ON COLUMN tenant_notification_preferences.created_at IS '创建时间';
COMMENT ON COLUMN tenant_notification_preferences.updated_at IS '更新时间';
COMMENT ON TABLE tenant_notification_preference_recipients IS '事件偏好接收规则关联（000039）：动态规则（事件发起人/事件受众/系统管理员）每偏好至多一条，自定义联系人不能重复关联；custom_recipient 仅进入邮件/短信投递，不产生站内收件箱';
COMMENT ON COLUMN tenant_notification_preference_recipients.id IS '自增主键';
COMMENT ON COLUMN tenant_notification_preference_recipients.tenant_id IS '归属租户 ID（与 preference_id 冗余一致，服务层写入前复核）';
COMMENT ON COLUMN tenant_notification_preference_recipients.preference_id IS '所属事件偏好行 ID';
COMMENT ON COLUMN tenant_notification_preference_recipients.target_kind IS '接收对象类型：event_actor/event_audience/tenant_admin/custom_recipient';
COMMENT ON COLUMN tenant_notification_preference_recipients.custom_recipient_id IS '自定义联系人 ID；仅 target_kind=custom_recipient 时必填（CHECK 强制组合合法）';
COMMENT ON COLUMN tenant_notification_preference_recipients.created_at IS '创建时间';
COMMENT ON TABLE tenant_notification_custom_recipients IS '租户自定义外部提醒对象池（000039）：仅用于邮件/短信渠道的外部联系人，上限默认 200（服务端配置控制）；软删除保留关联历史，删除前校验未被偏好引用';
COMMENT ON COLUMN tenant_notification_custom_recipients.id IS '自增主键（revision 删除口令）';
COMMENT ON COLUMN tenant_notification_custom_recipients.tenant_id IS '归属租户 ID';
COMMENT ON COLUMN tenant_notification_custom_recipients.name IS '姓名（trim 后 1–80 字符）';
COMMENT ON COLUMN tenant_notification_custom_recipients.mobile IS '规范化手机号；手机/邮箱至少一项必填（服务层校验）';
COMMENT ON COLUMN tenant_notification_custom_recipients.email IS '小写规范化邮箱；手机/邮箱至少一项必填（服务层校验）';
COMMENT ON COLUMN tenant_notification_custom_recipients.revision IS '联系人自身修改口令（一期新增/删除路径预留）';
COMMENT ON COLUMN tenant_notification_custom_recipients.created_at IS '创建时间';
COMMENT ON COLUMN tenant_notification_custom_recipients.updated_at IS '更新时间';
COMMENT ON COLUMN tenant_notification_custom_recipients.deleted_at IS '软删除时间，NULL=未删除；置位后不可再被偏好关联';
COMMENT ON TABLE notification_outbox_events IS '消息中心事务 Outbox（000039）：业务域在自身事务内写入结构化事件，随业务提交/回滚；Dispatcher Worker 领取后渲染扇出，重试以 event_id 幂等';
COMMENT ON COLUMN notification_outbox_events.id IS '自增主键（Worker 领取顺序）';
COMMENT ON COLUMN notification_outbox_events.event_id IS '生产者事件唯一 ID（全局唯一，幂等键）';
COMMENT ON COLUMN notification_outbox_events.tenant_id IS '归属租户 ID';
COMMENT ON COLUMN notification_outbox_events.event_code IS '事件注册表事件码';
COMMENT ON COLUMN notification_outbox_events.actor_member_id IS '事件发起成员 ID；0 表示系统发起';
COMMENT ON COLUMN notification_outbox_events.audience_member_ids IS '事件显式成员受众 JSONB 数组，Worker 扇出前批量复核同租户与有效状态';
COMMENT ON COLUMN notification_outbox_events.parameters IS '模板参数 JSONB（受事件注册表参数 Schema 限制）';
COMMENT ON COLUMN notification_outbox_events.occurred_at IS '业务事件发生时间';
COMMENT ON COLUMN notification_outbox_events.status IS '处理状态：pending/processing/done/failed';
COMMENT ON COLUMN notification_outbox_events.attempt_count IS '已尝试次数（有界指数退避，超上限进入 failed）';
COMMENT ON COLUMN notification_outbox_events.next_attempt_at IS '下次可领取时间（退避后的到期口令）';
COMMENT ON COLUMN notification_outbox_events.last_error_code IS '最近一次失败的稳定内部错误码（不存原始敏感错误）';
COMMENT ON COLUMN notification_outbox_events.created_at IS '事件写入时间（业务事务内）';
COMMENT ON COLUMN notification_outbox_events.processed_at IS '完成处理时间（done/failed 落位）';
