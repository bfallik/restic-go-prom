package restic

import (
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

type BackupMsgTyper interface {
	MsgType() string
}

func (r Repo) BackUp(contentDir string) ([]BackupMsgTyper, error) {
	cmd, err := NewCmd(CmdOptRepo{r.Config.Dir}, CmdOptCommandBackup{}, CmdOptFlags{"--insecure-no-password", contentDir, "--json"})
	if err != nil {
		return nil, err
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, cmd.ErrBuf.String())
	}

	lines, err := cmd.JSONLines()
	if err != nil {
		return nil, err
	}

	res := []BackupMsgTyper{}
	for _, msg := range lines {
		var typ ResticBackupMessageType
		if err := json.Unmarshal(msg, &typ); err != nil {
			return nil, err
		}

		switch typ.MessageType {
		case "status":
			var dest ResticBackupStatusJSON
			if err := json.Unmarshal(msg, &dest); err != nil {
				return nil, err
			}
			res = append(res, dest)

		case "summary":
			var dest ResticBackupSummaryJSON
			if err := json.Unmarshal(msg, &dest); err != nil {
				return nil, err
			}
			res = append(res, dest)

		default:
			panic("unexpected type")
		}
	}

	return res, nil
}

func (r Repo) Stats() (ResticStatsJSON, error) {
	cmd, err := NewCmd(CmdOptRepo{r.Config.Dir}, CmdOptCommandStats{}, CmdOptFlags{"--insecure-no-password", "--json"})
	if err != nil {
		return ResticStatsJSON{}, err
	}

	if err := cmd.Run(); err != nil {
		return ResticStatsJSON{}, fmt.Errorf("%w: %s", err, cmd.ErrBuf.String())
	}

	var dest ResticStatsJSON
	if err := json.Unmarshal(cmd.OutBuf.Bytes(), &dest); err != nil {
		return ResticStatsJSON{}, err
	}

	return dest, nil
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

type ResticBackupMessageType struct {
	MessageType string `json:"message_type"`
}

type ResticBackupStatusJSON struct {
	MessageType string `json:"message_type"`
	PercentDone int    `json:"percent_done"`
	TotalFiles  int    `json:"total_files"`
	FilesDone   int    `json:"files_done"`
	TotalBytes  int    `json:"total_bytes"`
	BytesDone   int    `json:"bytes_done"`
}

func (o ResticBackupStatusJSON) MsgType() string {
	return o.MessageType
}

type ResticBackupSummaryJSON struct {
	MessageType         string  `json:"message_type"`
	FilesNew            int     `json:"files_new"`
	FilesChanged        int     `json:"files_changed"`
	FilesUnmodified     int     `json:"files_unmodified"`
	DirsNew             int     `json:"dirs_new"`
	DirsChanges         int     `json:"dirs_changed"`
	DirsUnmodified      int     `json:"dirs_unmodified"`
	DataBlobs           int     `json:"data_blobs"`
	TreeBlobs           int     `json:"tree_blobs"`
	DataAdded           int     `json:"data_added"`
	DataAddedPacked     int     `json:"data_added_packed"`
	TotalFilesProcessed int     `json:"total_files_processed"`
	TotalBytesProcessed int     `json:"total_bytes_processed"`
	TotalDuration       float64 `json:"total_duration"`
	SnapshotID          string  `json:"snapshot_id"`
}

func (o ResticBackupSummaryJSON) MsgType() string {
	return o.MessageType
}
