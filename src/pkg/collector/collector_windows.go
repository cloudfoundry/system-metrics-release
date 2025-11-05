//go:build windows

package collector

import (
	"context"

	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/net"
)

func (s defaultRawCollector) AvgWithContext(ctx context.Context) (*load.AvgStat, error) {
	return &load.AvgStat{}, nil
}

func (s defaultRawCollector) ProtoCountersWithContext(ctx context.Context, protocols []string) ([]net.ProtoCountersStat, error) {
	return nil, nil
}
