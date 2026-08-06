//go:build linux

package cpu

import (
	"context"
	"path/filepath"
	"testing"

	gopsutilcommon "github.com/shirou/gopsutil/v4/common"
)

func TestGopsutilSourceReadsOnlyCPUAndLoadProcFixtures(t *testing.T) {
	t.Parallel()
	procPath, err := filepath.Abs(filepath.Join("testdata", "proc"))
	if err != nil {
		t.Fatalf("resolve proc fixture: %v", err)
	}
	ctx := context.WithValue(context.Background(), gopsutilcommon.EnvKey, gopsutilcommon.EnvMap{
		gopsutilcommon.HostProcEnvKey: procPath,
	})
	source := gopsutilSource{}
	counters, err := source.Counters(ctx)
	if err != nil {
		t.Fatalf("Counters() error = %v", err)
	}
	if len(counters) != 2 || counters[0].LogicalIndex != 0 || counters[1].LogicalIndex != 4 {
		t.Fatalf("counters = %+v", counters)
	}
	average, err := source.LoadAverage(ctx)
	if err != nil {
		t.Fatalf("LoadAverage() error = %v", err)
	}
	if average.One != 0.42 || average.Five != 0.84 || average.Fifteen != 1.26 {
		t.Fatalf("load average = %+v", average)
	}
}
