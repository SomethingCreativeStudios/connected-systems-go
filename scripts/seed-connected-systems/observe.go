package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"net/url"
	"strings"
	"time"
)

type Observer struct {
	cfg    Config
	client *APIClient
	rng    *rand.Rand
	report Report
}

type listedDatastream struct {
	ID                  string   `json:"id"`
	Formats             []string `json:"formats"`
	SamplingFeatureLink *apiLink `json:"samplingFeature@link"`
	ProcedureLink       *apiLink `json:"procedure@link"`
}

type apiLink struct {
	Href string `json:"href"`
}

type datastreamCollection struct {
	Items []listedDatastream `json:"items"`
	Links []collectionLink   `json:"links"`
}

type collectionLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func NewObserver(cfg Config, client *APIClient, rng *rand.Rand) *Observer {
	return &Observer{
		cfg:    cfg,
		client: client,
		rng:    rng,
		report: NewReport("observe", cfg.Namespace),
	}
}

func (o *Observer) Run(ctx context.Context) (Report, error) {
	eligible, skipped, err := o.refresh(ctx)
	o.report.Skipped += skipped
	if err != nil {
		return o.report, err
	}
	if len(eligible) == 0 {
		return o.report, fmt.Errorf("no compatible existing datastreams were found")
	}
	log.Printf("observation mode found %d compatible datastreams (%d skipped)", len(eligible), skipped)

	ticks := time.NewTicker(o.cfg.Observe.Frequency.Std())
	refreshes := time.NewTicker(o.cfg.Observe.StreamRefreshInterval.Std())
	defer ticks.Stop()
	defer refreshes.Stop()

	for {
		select {
		case <-ctx.Done():
			return o.report, nil
		case <-refreshes.C:
			updated, skipped, err := o.refresh(ctx)
			if err != nil {
				log.Printf("refresh eligible datastreams failed: %v", err)
				continue
			}
			o.report.Skipped += skipped
			if len(updated) == 0 {
				log.Printf("refresh found no compatible datastreams; retaining %d cached streams", len(eligible))
				continue
			}
			eligible = updated
			log.Printf("refreshed eligible datastreams: %d compatible, %d skipped", len(eligible), skipped)
		case <-ticks.C:
			batch := o.cfg.Observe.BatchSize.Min
			if o.cfg.Observe.BatchSize.Max > o.cfg.Observe.BatchSize.Min {
				batch += o.rng.IntN(o.cfg.Observe.BatchSize.Max - o.cfg.Observe.BatchSize.Min + 1)
			}
			for _, stream := range chooseSubset(o.rng, eligible, batch) {
				if err := o.sendObservation(ctx, stream); err != nil {
					o.report.AddFailed()
					log.Printf("observation for datastream %s failed: %v", stream.ID, err)
					continue
				}
				o.report.AddSent()
			}
		}
	}
}

func (o *Observer) refresh(ctx context.Context) ([]seededDatastream, int, error) {
	listed, err := o.listDatastreams(ctx)
	if err != nil {
		return nil, 0, err
	}
	eligible := make([]seededDatastream, 0, len(listed))
	skipped := 0
	for _, datastream := range listed {
		stream, ok := o.loadGeneratableStream(ctx, datastream)
		if !ok {
			skipped++
			continue
		}
		eligible = append(eligible, stream)
	}
	return eligible, skipped, nil
}

func (o *Observer) listDatastreams(ctx context.Context) ([]listedDatastream, error) {
	next := "/datastreams?limit=100"
	items := make([]listedDatastream, 0)
	seen := map[string]bool{}
	for next != "" {
		if seen[next] {
			return nil, fmt.Errorf("datastream pagination repeated %q", next)
		}
		seen[next] = true
		var page datastreamCollection
		if err := o.client.GetJSON(ctx, next, &page); err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		next = ""
		for _, link := range page.Links {
			if strings.EqualFold(link.Rel, "next") && link.Href != "" {
				next = link.Href
				break
			}
		}
	}
	return items, nil
}

func (o *Observer) loadGeneratableStream(ctx context.Context, listed listedDatastream) (seededDatastream, bool) {
	if listed.ID == "" {
		return seededDatastream{}, false
	}
	formats := listed.Formats
	if len(formats) == 0 {
		formats = []string{"application/json"}
	}
	for _, format := range formats {
		resourcePath := "/datastreams/" + url.PathEscape(listed.ID) + "/schema?obsFormat=" + url.QueryEscape(format)
		var schema StreamSchema
		if err := o.client.GetJSON(ctx, resourcePath, &schema); err != nil {
			continue
		}
		if _, err := resultForSchema(o.rng, schema); err != nil {
			continue
		}
		stream := seededDatastream{ID: listed.ID, Schema: schema}
		if listed.SamplingFeatureLink != nil {
			stream.SamplingFeatureID, _ = resourceID(listed.SamplingFeatureLink.Href, "samplingFeatures")
		}
		if listed.ProcedureLink != nil {
			stream.ProcedureID, _ = resourceID(listed.ProcedureLink.Href, "procedures")
		}
		return stream, true
	}
	return seededDatastream{}, false
}

func (o *Observer) sendObservation(ctx context.Context, stream seededDatastream) error {
	result, err := resultForSchema(o.rng, stream.Schema)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Truncate(time.Second)
	payload := map[string]any{
		"resultTime":     now.Format(time.RFC3339),
		"phenomenonTime": now.Add(-5 * time.Second).Format(time.RFC3339),
		"result":         result,
		"parameters":     map[string]any{"source": "seed-connected-systems", "qc": "good"},
	}
	if stream.SamplingFeatureID != "" {
		payload["samplingFeature@id"] = stream.SamplingFeatureID
	}
	if stream.ProcedureID != "" {
		payload["procedure@link"] = map[string]any{"href": o.client.URL("/procedures/" + stream.ProcedureID)}
	}
	resourcePath := "/datastreams/" + url.PathEscape(stream.ID) + "/observations"
	response, err := o.client.PostJSON(ctx, resourcePath, payload)
	if err != nil {
		return err
	}
	if _, err := expectCreated(o.client, resourcePath, "observations", response); err != nil {
		return err
	}
	return nil
}

// chooseSubset handles both identifiers and richer values without mutating the
// cached source slice.
func chooseSubset[T any](rng *rand.Rand, values []T, count int) []T {
	if count > len(values) {
		count = len(values)
	}
	if count <= 0 {
		return []T{}
	}
	permutation := rng.Perm(len(values))
	selected := make([]T, 0, count)
	for _, index := range permutation[:count] {
		selected = append(selected, values[index])
	}
	return selected
}
