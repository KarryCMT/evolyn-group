DROP INDEX IF EXISTS uk_tenant_public_invitation_links_token;
DROP INDEX IF EXISTS uk_tenant_public_invitation_links_tenant;
DROP TABLE IF EXISTS tenant_public_invitation_links;
DROP INDEX IF EXISTS idx_member_invitations_tenant_status;
DROP INDEX IF EXISTS uk_member_invitations_tenant_email_pending;
DROP INDEX IF EXISTS uk_member_invitations_tenant_phone_pending;
DROP INDEX IF EXISTS uk_member_invitations_token;
DROP TABLE IF EXISTS member_invitations;
