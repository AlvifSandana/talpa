package analyze

import (
	"context"
	"os"
	"sort"
	"strconv"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/infra/filesystem"
)

type Service struct{}

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext, root string, depth int, limit int) (model.CommandResult, error) {
	_ = ctx
	start := time.Now()
	if root == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return model.CommandResult{}, err
		}
		root = h
	}

	items, err := filesystem.Scan(root, filesystem.ScanOptions{MaxDepth: depth, Excludes: []string{"/proc", "/sys", "/dev", "/run"}})
	if err != nil {
		return model.CommandResult{}, err
	}

	sort.Slice(items, func(i, j int) bool { return items[i].SizeBytes > items[j].SizeBytes })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	out := make([]model.CandidateItem, 0, len(items))
	var estimate int64
	for i, it := range items {
		estimate += it.SizeBytes
		out = append(out, model.CandidateItem{
			ID:           "analyze-" + strconv.Itoa(i+1),
			RuleID:       "",
			Path:         it.Path,
			SizeBytes:    it.SizeBytes,
			LastModified: it.LastModified,
			Category:     "tree_node",
			Risk:         model.RiskMedium,
			Selected:     false,
			RequiresRoot: false,
			Result:       "planned",
		})
	}

	return model.CommandResult{
		SchemaVersion: "1.0",
		Command:       "analyze",
		Timestamp:     time.Now().UTC(),
		DurationMS:    time.Since(start).Milliseconds(),
		DryRun:        app.Options.DryRun,
		Summary: model.Summary{
			ItemsTotal:          len(out),
			ItemsSelected:       0,
			EstimatedFreedBytes: estimate,
			Errors:              0,
		},
		Items: out,
	}, nil
}
