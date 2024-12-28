package restic

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

func TestMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()

	m := NewMetrics(reg)
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

func NewTempRepo(tmpDir TestTempDir) (*Repo, error) {
	contentDir := path.Join(tmpDir.Path, "content")
	repoDir := path.Join(tmpDir.Path, "repo")

	if err := os.Mkdir(contentDir, 0755); err != nil {
		return nil, err
	}
	if err := os.Mkdir(repoDir, 0755); err != nil {
		return nil, err
	}

	for n, bs := range [][]byte{
		[]byte("This is a test"),
		{},                      // empty
		[]byte("DFGDFGDGDFGFD"), // BFTODO: random
	} {
		fname := fmt.Sprintf("file_%d", n)
		if err := os.WriteFile(path.Join(contentDir, fname), bs, 0644); err != nil {
			return nil, err
		}
	}

	repo := NewRepo(Config{Dir: repoDir})

	if err := repo.Init(); err != nil {
		return nil, err
	}

	if err := repo.BackUp(contentDir); err != nil {
		return nil, err
	}

	return repo, nil
}

func TestRepo(t *testing.T) {
	tmpDir, err := NewTestTempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer tmpDir.Close()
	slog.Info("temporary dir", slog.String("path", tmpDir.Path))

	repo, err := NewTempRepo(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	stats, err := repo.Stats()
	if err != nil {
		t.Fatal(err)
	}

	const ExpTotalSize = 27
	const ExpTotalFileCount = 10
	const ExpSnapshotsCount = 1

	if ExpTotalSize != stats.TotalSize {
		t.Errorf("unexpected total size, got %d, expected %d", stats.TotalSize, ExpTotalSize)
	}
	if ExpTotalFileCount != stats.TotalFileCount {
		t.Errorf("unexpected total file count, got %d, expected %d", stats.TotalFileCount, ExpTotalFileCount)
	}
	if ExpSnapshotsCount != stats.SnapshotsCount {
		t.Errorf("unexpected snapshots count, got %d, expected %d", stats.SnapshotsCount, ExpSnapshotsCount)
	}
}
