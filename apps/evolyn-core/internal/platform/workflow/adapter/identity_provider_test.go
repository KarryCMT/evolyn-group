package adapter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 实际调用身份端口并编译 SQL，防止聚合查询退回旧表名或丢失租户条件。
func TestIdentityMemberQueriesUseTenantMemberTable(t *testing.T) {
	for _, operation := range []string{"display-name", "context"} {
		t.Run(operation, func(t *testing.T) {
			db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost"}), &gorm.Config{DisableAutomaticPing: true, DryRun: true})
			require.NoError(t, err)
			var queries []string
			var arguments [][]any
			require.NoError(t, db.Callback().Query().After("gorm:query").Register("capture_identity_sql", func(tx *gorm.DB) {
				queries = append(queries, tx.Statement.SQL.String())
				arguments = append(arguments, append([]any(nil), tx.Statement.Vars...))
			}))
			identity := NewIdentityProvider(db)
			if operation == "display-name" {
				identity.MemberDisplayName(context.Background(), 106, 49)
			} else {
				_, _, err = identity.MemberContext(context.Background(), 106, 49)
				require.NoError(t, err)
			}
			require.NotEmpty(t, queries)
			require.Contains(t, queries[0], `FROM "tn_users"`)
			require.NotContains(t, queries[0], `FROM "users"`)
			require.Contains(t, queries[0], "pf_accounts.id = tn_users.account_id")
			require.Contains(t, queries[0], "tn_users.id = $1 AND tn_users.tenant_id = $2")
			require.Equal(t, []any{uint(49), uint(106), 1}, arguments[0])
		})
	}
}
