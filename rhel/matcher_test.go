package rhel

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/quay/zlog"

	"github.com/quay/claircore"
	"github.com/quay/claircore/internal/matcher"
	"github.com/quay/claircore/internal/updater"
	vulnstore "github.com/quay/claircore/internal/vulnstore/postgres"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/test"
	"github.com/quay/claircore/test/integration"
)

func TestMatcherIntegration(t *testing.T) {
	integration.Skip(t)
	ctx := zlog.Test(context.Background(), t)
	pool := vulnstore.TestDB(ctx, t)
	store := vulnstore.NewVulnStore(pool)
	m := &Matcher{}
	fs, err := filepath.Glob("testdata/*.xml")
	if err != nil {
		t.Error(err)
	}

	ch := make(chan driver.Updater)
	go func() {
		for _, f := range fs {
			u, err := test.Updater(f)
			if err != nil {
				t.Error(err)
				continue
			}
			ch <- u
		}
		close(ch)
	}()
	exec := updater.Online{Pool: pool}

	// force update
	if err := exec.Run(ctx, ch); err != nil {
		t.Error(err)
	}

	f, err := os.Open(filepath.Join("testdata", "rhel-report.json"))
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer f.Close()
	var ir claircore.IndexReport
	if err := json.NewDecoder(f).Decode(&ir); err != nil {
		t.Fatalf("failed to decode IndexReport: %v", err)
	}
	vr, err := matcher.Match(ctx, &ir, []driver.Matcher{m}, store)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(ioutil.Discard).Encode(&vr); err != nil {
		t.Fatalf("failed to marshal VR: %v", err)
	}
}
