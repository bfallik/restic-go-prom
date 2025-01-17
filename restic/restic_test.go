package restic

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

func TestMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()

	m := NewMetrics()
	reg.Register(m.CheckSuccess)
	m.CheckSuccess.Set(1.1)

	gatherers := prometheus.Gatherers{
		reg,
	}

	gathering, err := gatherers.Gather()
	if err != nil {
		t.Fatal()
	}

	out := &bytes.Buffer{}
	for _, mf := range gathering {
		if _, err := expfmt.MetricFamilyToText(out, mf); err != nil {
			t.Fatal(err)
		}
	}

	const exp = `# HELP restic_check_success Result of restic check operation in the repository
# TYPE restic_check_success gauge
restic_check_success 1.1
`

	if out.String() != exp {
		t.Errorf("mismatched output, expected '%s', got '%s'", exp, out.String())
	}
}

type TestTempDir struct {
	Path string
	Repo *Repo
}

func NewTestTempDir() (TestTempDir, error) {
	tmpDir, err := os.MkdirTemp("", "restic-go-prom-test-dir")
	if err != nil {
		return TestTempDir{}, err
	}
	return TestTempDir{
		Path: tmpDir,
	}, nil
}

func (t *TestTempDir) Close() error {
	disableCleanup := os.Getenv("DISABLE_CLEANUP")
	if disableCleanup != "" {
		slog.Info("temporary dir cleanup disabled", slog.String("path", t.Path))
		return nil
	}
	return os.RemoveAll(t.Path)
}

func randBytes(len int) []byte {
	bs := make([]byte, len)
	rand.Read(bs)
	return bs
}

func NewTempRepo(t *testing.T, tmpDir TestTempDir) (*Repo, ResticBackupSummaryJSON, error) {
	contentDir := path.Join(tmpDir.Path, "content")
	repoDir := path.Join(tmpDir.Path, "repo")

	if err := os.Mkdir(contentDir, 0755); err != nil {
		return nil, ResticBackupSummaryJSON{}, err
	}
	if err := os.Mkdir(repoDir, 0755); err != nil {
		return nil, ResticBackupSummaryJSON{}, err
	}

	fileData := [][]byte{
		[]byte("This is a test"), // fixed string
		{},                       // empty
		randBytes(40),            // random
	}
	for n, bs := range fileData {
		fname := fmt.Sprintf("file_%d", n)
		if err := os.WriteFile(path.Join(contentDir, fname), bs, 0644); err != nil {
			return nil, ResticBackupSummaryJSON{}, err
		}
	}

	repo := NewRepo(Config{Dir: repoDir})

	if err := repo.Init(); err != nil {
		return nil, ResticBackupSummaryJSON{}, err
	}

	msgs, err := repo.BackUp(contentDir)
	if err != nil {
		return nil, ResticBackupSummaryJSON{}, err
	}

	summary := ResticBackupSummaryJSON{}
	for _, msg := range msgs {
		switch v := msg.(type) {
		case ResticBackupStatusJSON:
			t.Logf("status: %v", v)
		case ResticBackupSummaryJSON:
			summary = v
		default:
			panic("unexpected type")
		}
	}

	if summary.FilesNew != len(fileData) {
		return nil, ResticBackupSummaryJSON{}, fmt.Errorf("FilesNew mismatch; expected %v, got %v", len(fileData), summary.FilesNew)
	}
	tot := 0
	for _, bs := range fileData {
		tot += len(bs)
	}

	if summary.TotalBytesProcessed != tot {
		return nil, ResticBackupSummaryJSON{}, fmt.Errorf("DataAdded mismatch; expected %v, got %v", tot, summary.TotalBytesProcessed)
	}

	return repo, summary, nil
}

func TestRepo(t *testing.T) {
	tmpDir, err := NewTestTempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer tmpDir.Close()
	slog.Info("temporary dir", slog.String("path", tmpDir.Path))

	repo, summary, err := NewTempRepo(t, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	stats, err := repo.Stats()
	if err != nil {
		t.Fatal(err)
	}

	expected := ResticStatsJSON{
		TotalSize:      summary.TotalBytesProcessed,
		TotalFileCount: summary.FilesNew + summary.DirsNew,
		SnapshotsCount: 1,
	}

	if diff := cmp.Diff(expected, stats); diff != "" {
		t.Errorf("Stats() mismatch (-expected, +actual):\n%s", diff)
	}
}
