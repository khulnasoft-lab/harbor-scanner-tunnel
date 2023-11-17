package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/etc"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/harbor"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/job"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/persistence"
	redis "github.com/redis/go-redis/v9"
	"golang.org/x/xerrors"
)

type store struct {
	cfg etc.RedisStore
	rdb *redis.Client
}

func NewStore(cfg etc.RedisStore, rdb *redis.Client) persistence.Store {
	return &store{cfg: cfg, rdb: rdb}
}

func (s *store) Create(ctx context.Context, scanJob job.ScanJob) error {
	bytes, err := json.Marshal(scanJob)
	if err != nil {
		return xerrors.Errorf("marshalling scan job: %w", err)
	}

	key := s.keyForScanJob(scanJob.ID)

	slog.Debug("Saving scan job",
		slog.String("scan_job_id", scanJob.ID),
		slog.String("scan_job_status", scanJob.Status.String()),
		slog.String("redis_key", key),
		slog.Duration("expire", s.cfg.ScanJobTTL),
	)

	if err = s.rdb.SetNX(ctx, key, string(bytes), s.cfg.ScanJobTTL).Err(); err != nil {
		return xerrors.Errorf("creating scan job: %w", err)
	}

	return nil
}

func (s *store) update(ctx context.Context, scanJob job.ScanJob) error {
	bytes, err := json.Marshal(scanJob)
	if err != nil {
		return xerrors.Errorf("marshalling scan job: %w", err)
	}

	key := s.keyForScanJob(scanJob.ID)

	slog.Debug("Updating scan job",
		slog.String("scan_job_id", scanJob.ID),
		slog.String("scan_job_status", scanJob.Status.String()),
		slog.String("redis_key", key),
		slog.Duration("expire", s.cfg.ScanJobTTL),
	)

	if err = s.rdb.SetXX(ctx, key, string(bytes), s.cfg.ScanJobTTL).Err(); err != nil {
		return xerrors.Errorf("updating scan job: %w", err)
	}

	return nil
}

func (s *store) Get(ctx context.Context, scanJobID string) (*job.ScanJob, error) {
	key := s.keyForScanJob(scanJobID)
	value, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var scanJob job.ScanJob
	if err = json.Unmarshal([]byte(value), &scanJob); err != nil {
		return nil, xerrors.Errorf("unmarshalling scan job: %w", err)
	}

	return &scanJob, nil
}

func (s *store) UpdateStatus(ctx context.Context, scanJobID string, newStatus job.ScanJobStatus, error ...string) error {
	slog.Debug("Updating status for scan job", slog.String("scan_job_id", scanJobID),
		slog.String("new_status", newStatus.String()),
	)

	scanJob, err := s.Get(ctx, scanJobID)
	if scanJob == nil {
		return xerrors.Errorf("scan job %s not found", scanJobID)
	} else if err != nil {
		return err
	}

	scanJob.Status = newStatus
	if len(error) > 0 {
		scanJob.Error = error[0]
	}

	return s.update(ctx, *scanJob)
}

func (s *store) UpdateReport(ctx context.Context, scanJobID string, report harbor.ScanReport) error {
	slog.Debug("Updating reports for scan job", slog.String("scan_job_id", scanJobID))

	scanJob, err := s.Get(ctx, scanJobID)
	if err != nil {
		return err
	}

	scanJob.Report = report
	return s.update(ctx, *scanJob)
}

func (s *store) keyForScanJob(scanJobID string) string {
	return fmt.Sprintf("%s:scan-job:%s", s.cfg.Namespace, scanJobID)
}
