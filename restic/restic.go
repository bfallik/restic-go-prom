package restic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	Dir string
}

type Metrics struct {
	CheckSuccess         prometheus.Gauge
	LocksTotal           prometheus.Counter
	SnapshotsTotal       prometheus.Counter
	BackupTimestamp      prometheus.Gauge
	BackupFilesTotal     prometheus.Counter
	BackupSizeTotal      prometheus.Counter
	BackupSnapshotsTotal prometheus.Counter
	ScrapeDurationSecs   prometheus.Gauge
}

func NewMetrics() *Metrics {
	m := Metrics{
		CheckSuccess: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "restic_check_success",
				Help: "Result of restic check operation in the repository",
			},
		),
		LocksTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "restic_locks_total",
				Help: "Total number of locks in the repository",
			},
		),
		SnapshotsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "restic_snapshots_total",
				Help: "Total number of snapshots in the repository",
			},
		),
		BackupTimestamp: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "restic_backup_timestamp",
				Help: "Timestamp of the last backup",
			},
		),

		BackupFilesTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "restic_backup_files_total",
				Help: "Number of files in the backup",
			},
		),

		BackupSizeTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "restic_backup_size_total",
				Help: "Total size of backup in bytes",
			},
		),
		BackupSnapshotsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "restic_backup_snapshots_total",
				Help: "Total number of snapshots",
			},
		),
		ScrapeDurationSecs: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "restic_scrape_duration_secs",
				Help: "Amount of time each scrape takes",
			},
		),
	}

	return &m
}

func (m *Metrics) MustRegister(reg *prometheus.Registry) {
	reg.MustRegister(
		m.CheckSuccess,
		m.LocksTotal,
		m.SnapshotsTotal,
		m.BackupTimestamp,
		m.BackupFilesTotal,
		m.BackupSizeTotal,
		m.BackupSnapshotsTotal,
		m.ScrapeDurationSecs,
	)
}

type Repo struct {
	Config Config
}

func NewRepo(cfg Config) *Repo {
	return &Repo{
		Config: cfg,
	}
}

func (r Repo) Init() error {
	cmd := exec.Command("restic", "init", "--repo", r.Config.Dir, "--insecure-no-password")
	if cmd.Err != nil {
		return cmd.Err
	}
	return cmd.Run()
}

func (r Repo) BackUp(contentDir string) error {
	cmd := exec.Command("restic", "backup", "--repo", r.Config.Dir, "--insecure-no-password", contentDir)
	if cmd.Err != nil {
		return cmd.Err
	}
	return cmd.Run()
}

func SubcommandJSON[T any](repo Repo, subCmd string, arg ...string) (T, error) {
	var zero T

	cmd := exec.Command("restic", append([]string{subCmd, "--repo", repo.Config.Dir, "--json"}, arg...)...)
	if cmd.Err != nil {
		return zero, cmd.Err
	}

	outBuf := bytes.Buffer{}
	cmd.Stdout = &outBuf

	errBuf := bytes.Buffer{}
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return zero, fmt.Errorf("%w: %s", err, errBuf.String())
	}

	var dest T
	if err := json.Unmarshal(outBuf.Bytes(), &dest); err != nil {
		return zero, err
	}

	return dest, nil
}

func (r Repo) Stats() (ResticStatsJSON, error) {
	return SubcommandJSON[ResticStatsJSON](r, "stats", "--insecure-no-password")
}

func (r *Repo) Ping() error {
	return nil
}

func (r *Repo) Metrics() Metrics {
	return Metrics{}
}

type ResticStatsJSON struct {
	TotalSize      int `json:"total_size"`
	TotalFileCount int `json:"total_file_count"`
	SnapshotsCount int `json:"snapshots_count"`
}
