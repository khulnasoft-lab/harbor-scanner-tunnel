package persistence

import (
	"context"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/harbor"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/job"
)

type Store interface {
	Create(ctx context.Context, scanJob job.ScanJob) error
	Get(ctx context.Context, scanJobID string) (*job.ScanJob, error)
	UpdateStatus(ctx context.Context, scanJobID string, newStatus job.ScanJobStatus, error ...string) error
	UpdateReport(ctx context.Context, scanJobID string, report harbor.ScanReport) error
}
