package memory

import (
	"context"
	"fmt"
	"os"
)

type Source interface {
	MemInfo(context.Context) ([]byte, error)
	Pressure(context.Context) ([]byte, error)
}

type procSource struct{}

func (procSource) MemInfo(ctx context.Context) ([]byte, error) {
	return readProcFile(ctx, "/proc/meminfo")
}

func (procSource) Pressure(ctx context.Context) ([]byte, error) {
	return readProcFile(ctx, "/proc/pressure/memory")
}

func readProcFile(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}
