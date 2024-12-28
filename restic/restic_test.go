package restic

import (
	"bytes"
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
