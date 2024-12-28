package restic

import (
	"bytes"
	"encoding/json"
	"os/exec"

	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	Dir string
}

type Metrics struct {
	CheckSuccess prometheus.Gauge
	/*
	   	# HELP restic_check_success Result of restic check operation in the repository

	   # TYPE restic_check_success gauge
	   restic_check_success 1.0
	   # HELP restic_locks_total Total number of locks in the repository
	   # TYPE restic_locks_total counter
	   restic_locks_total 1.0
	   # HELP restic_snapshots_total Total number of snapshots in the repository
	   # TYPE restic_snapshots_total counter
	   restic_snapshots_total 100.0
	   # HELP restic_backup_timestamp Timestamp of the last backup
	   # TYPE restic_backup_timestamp gauge
	   restic_backup_timestamp{client_hostname="product.example.com",client_username="root",client_version="restic 0.16.0",snapshot_hash="20795072cba0953bcdbe52e9cf9d75e5726042f5bbf2584bb2999372398ee835",snapshot_tag="mysql",snapshot_tags="mysql,tag2",snapshot_paths="/mysql/data,/mysql/config"} 1.666273638e+09
	   # HELP restic_backup_files_total Number of files in the backup
	   # TYPE restic_backup_files_total counter
	   restic_backup_files_total{client_hostname="product.example.com",client_username="root",client_version="restic 0.16.0",snapshot_hash="20795072cba0953bcdbe52e9cf9d75e5726042f5bbf2584bb2999372398ee835",snapshot_tag="mysql",snapshot_tags="mysql,tag2",snapshot_paths="/mysql/data,/mysql/config"} 8.0
	   # HELP restic_backup_size_total Total size of backup in bytes
	   # TYPE restic_backup_size_total counter
	   restic_backup_size_total{client_hostname="product.example.com",client_username="root",client_version="restic 0.16.0",snapshot_hash="20795072cba0953bcdbe52e9cf9d75e5726042f5bbf2584bb2999372398ee835",snapshot_tag="mysql",snapshot_tags="mysql,tag2",snapshot_paths="/mysql/data,/mysql/config"} 4.3309562e+07
	   # HELP restic_backup_snapshots_total Total number of snapshots
	   # TYPE restic_backup_snapshots_total counter
	   restic_backup_snapshots_total{client_hostname="product.example.com",client_username="root",client_version="restic 0.16.0",snapshot_hash="20795072cba0953bcdbe52e9cf9d75e5726042f5bbf2584bb2999372398ee835",snapshot_tag="mysql",snapshot_tags="mysql,tag2",snapshot_paths="/mysql/data,/mysql/config"} 1.0
	   # HELP restic_scrape_duration_seconds Amount of time each scrape takes
	   # TYPE restic_scrape_duration_seconds gauge
	   restic_scrape_duration_seconds 166.9411084651947
	*/
}

func NewMetrics(reg *prometheus.Registry) *Metrics {
	m := Metrics{
		CheckSuccess: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "restic_check_success",
				Help: "Result of restic check operation in the repository",
			},
		),
	}
	reg.MustRegister(m.CheckSuccess)
	return &m
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

func (r Repo) Stats() (ResticStatsJSON, error) {
	cmd := exec.Command("restic", "stats", "--repo", r.Config.Dir, "--insecure-no-password", "--json")
	if cmd.Err != nil {
		return ResticStatsJSON{}, cmd.Err
	}

	outBuf := bytes.Buffer{}
	cmd.Stdout = &outBuf

	if err := cmd.Run(); err != nil {
		return ResticStatsJSON{}, err
	}

	stats := ResticStatsJSON{}
	if err := json.Unmarshal(outBuf.Bytes(), &stats); err != nil {
		return ResticStatsJSON{}, err
	}

	return stats, nil
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
