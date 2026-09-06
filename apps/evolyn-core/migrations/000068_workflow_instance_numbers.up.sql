-- 独立流程单号：租户内按东八区日期递增，与实例主键完全分离。
CREATE TABLE wf_instance_number_counters (
    tenant_id bigint NOT NULL,
    number_date varchar(8) NOT NULL,
    last_value bigint NOT NULL CHECK (last_value > 0),
    PRIMARY KEY (tenant_id, number_date)
);
ALTER TABLE wf_instance ADD COLUMN instance_no varchar(40);
ALTER TABLE tn_form_records ADD COLUMN workflow_instance_no varchar(40) NOT NULL DEFAULT '';

-- 历史实例按创建时间分配单独流水；主键仅用于同一时间的稳定排序。
WITH numbered AS (
    SELECT id, tenant_id,
           to_char(COALESCE(created_at, CURRENT_TIMESTAMP) AT TIME ZONE 'Asia/Shanghai', 'YYYYMMDD') AS day,
           row_number() OVER (PARTITION BY tenant_id, (COALESCE(created_at, CURRENT_TIMESTAMP) AT TIME ZONE 'Asia/Shanghai')::date ORDER BY created_at, id) AS seq
    FROM wf_instance
)
UPDATE wf_instance i SET instance_no = 'WF-' || n.day || '-' ||
    lpad(n.seq::text, greatest(6, length(n.seq::text)), '0')
FROM numbered n WHERE i.id = n.id;
INSERT INTO wf_instance_number_counters (tenant_id, number_date, last_value)
SELECT tenant_id, split_part(instance_no, '-', 2), max(split_part(instance_no, '-', 3)::bigint)
FROM wf_instance GROUP BY tenant_id, split_part(instance_no, '-', 2);
ALTER TABLE wf_instance ALTER COLUMN instance_no SET NOT NULL;
CREATE UNIQUE INDEX ux_wf_instance_tenant_number ON wf_instance(tenant_id, instance_no);

-- 有历史多次发起的记录绑定最早实例；没有实例的记录保留空值，不伪造流程。
WITH bindings AS (
 SELECT DISTINCT ON (tenant_id, form_id, business_id) tenant_id, form_id, form_version_id, business_id, instance_no
 FROM wf_instance WHERE form_id > 0
 ORDER BY tenant_id, form_id, business_id, created_at, id
)
UPDATE tn_form_records r SET workflow_instance_no = b.instance_no
FROM bindings b WHERE r.tenant_id = b.tenant_id AND r.form_id = b.form_id
AND r.form_version_id = b.form_version_id AND r.id::text = b.business_id;
CREATE UNIQUE INDEX ux_form_record_workflow_number ON tn_form_records(tenant_id, workflow_instance_no) WHERE workflow_instance_no <> '';
COMMENT ON COLUMN wf_instance.instance_no IS '不可变流程单号：WF-东八区日期-至少六位租户日流水';
COMMENT ON COLUMN tn_form_records.workflow_instance_no IS '只读流程单号系统字段，与首次提交创建的实例单号一致';
