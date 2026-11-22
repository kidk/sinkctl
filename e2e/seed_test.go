//go:build e2e

// Package e2e runs against a real Bugsink. Needs BUGSINK_URL and BUGSINK_TOKEN;
// `task e2e` starts a pinned container and sets both.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kidk/sinkctl/internal/bugsink"
	"github.com/kidk/sinkctl/internal/config"
	"github.com/kidk/sinkctl/internal/ops"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// data created once by TestMain and read by every test
var seed struct {
	ops        *ops.Ops
	project    int
	issue      string // UUID
	friendlyID string
	event      string // Bugsink event UUID
	sdkEventID string
}

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	err := seedData(ctx)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e seed:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func seedData(ctx context.Context) error {
	c := config.Context{URL: os.Getenv("BUGSINK_URL"), Token: os.Getenv("BUGSINK_TOKEN")}
	if c.URL == "" || c.Token == "" {
		return errors.New("BUGSINK_URL and BUGSINK_TOKEN must be set")
	}
	api, err := ops.NewClient(c, httpClient)
	if err != nil {
		return err
	}
	if seed.ops, err = ops.New(c, httpClient); err != nil {
		return err
	}

	name := "e2e-" + uuid.NewString()[:8]
	team, err := api.TeamsCreateWithResponse(ctx, bugsink.TeamCreateUpdate{Name: name})
	if err != nil {
		return err
	}
	if team.JSON201 == nil || team.JSON201.Id == nil {
		return fmt.Errorf("create team: %d %s", team.StatusCode(), team.Body)
	}
	if seed.project, err = createProject(ctx, api, name, *team.JSON201.Id); err != nil {
		return err
	}
	p, err := seed.ops.GetProject(ctx, ops.GetProjectIn{ID: seed.project})
	if err != nil {
		return err
	}
	if p.Dsn == nil {
		return errors.New("seeded project has no DSN")
	}
	seed.sdkEventID = uuid.NewString()
	if err := storeEvent(ctx, *p.Dsn, seed.sdkEventID); err != nil {
		return err
	}
	return waitForIssue(ctx)
}

// createProject skips the generated decoder: Bugsink 2.6.3 documents the created id as a UUID but returns "1".
func createProject(ctx context.Context, api *bugsink.ClientWithResponses, name string, team uuid.UUID) (int, error) {
	res, err := api.ProjectsCreate(ctx, bugsink.ProjectCreate{Name: name, Team: team})
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	var created struct {
		ID int `json:"id,string"`
	}
	if res.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("create project: %s", res.Status)
	}
	b, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(b, &created); err != nil {
		return 0, fmt.Errorf("create project: %w: %s", err, b)
	}
	return created.ID, nil
}

// storeEvent posts one Python-style exception via the Sentry store endpoint.
func storeEvent(ctx context.Context, dsn, eventID string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return err
	}
	ev, err := json.Marshal(map[string]any{
		"event_id":  strings.ReplaceAll(eventID, "-", ""),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"platform":  "python",
		"level":     "error",
		"exception": map[string]any{"values": []any{map[string]any{
			"type":       "ValueError",
			"value":      "e2e boom",
			"stacktrace": map[string]any{"frames": []any{map[string]any{"filename": "app.py", "function": "main", "lineno": 3}}},
		}}},
	})
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/api/%d/store/?sentry_key=%s", os.Getenv("BUGSINK_URL"), seed.project, u.User.Username())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(ev))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("store event: %s", res.Status)
	}
	return nil
}

// waitForIssue polls until the event is digested into an issue.
func waitForIssue(ctx context.Context) error {
	for range 30 {
		page, err := seed.ops.ListIssues(ctx, ops.ListIssuesIn{Project: seed.project})
		if err != nil {
			return err
		}
		if len(page.Results) > 0 {
			i := page.Results[0]
			if i.Id == nil || i.FriendlyId == nil {
				return fmt.Errorf("seeded issue lacks id: %+v", i)
			}
			seed.issue, seed.friendlyID = i.Id.String(), *i.FriendlyId
			events, err := seed.ops.ListEvents(ctx, ops.ListEventsIn{Issue: seed.issue})
			if err != nil {
				return err
			}
			if len(events.Results) == 0 || events.Results[0].Id == nil {
				return errors.New("seeded issue has no events")
			}
			seed.event = events.Results[0].Id.String()
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("seeded event never became an issue")
}
