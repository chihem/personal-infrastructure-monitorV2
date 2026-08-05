package audit

import (
	"context"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/migrations"
)

func Open(ctx context.Context, path string) (*storage.Database, error) {
	return storage.Open(ctx, path, migrations.Audit, storage.ReadWrite)
}
