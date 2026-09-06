DROP INDEX ux_form_record_workflow_number;
ALTER TABLE tn_form_records DROP COLUMN workflow_instance_no;
DROP INDEX ux_wf_instance_tenant_number;
ALTER TABLE wf_instance DROP COLUMN instance_no;
DROP TABLE wf_instance_number_counters;
