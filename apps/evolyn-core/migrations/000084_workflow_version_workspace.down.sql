ALTER TABLE wf_definition
    DROP CONSTRAINT IF EXISTS chk_wf_definition_version_workspace;

ALTER TABLE wf_definition
    DROP COLUMN IF EXISTS has_draft,
    DROP COLUMN IF EXISTS draft_version_no;
