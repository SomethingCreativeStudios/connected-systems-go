package pubsub

import (
	"context"
	"fmt"
	"strings"
)

func (r *RepositoryChangeResolver) SystemPath(ctx context.Context, id string) (string, error) {
	var chain []string
	seen := map[string]bool{}
	for id != "" {
		if seen[id] || len(chain) >= 256 {
			return "", fmt.Errorf("cyclic or excessively deep system ownership")
		}
		seen[id] = true
		system, err := r.repos.System.GetByID(ctx, id)
		if err != nil {
			return "", err
		}
		chain = append(chain, EncodeID(system.ID))
		id = ""
		if system.ParentSystemID != nil {
			id = *system.ParentSystemID
		}
	}
	if len(chain) == 0 {
		return "", fmt.Errorf("owning system is required")
	}
	path := "systems/" + chain[len(chain)-1]
	for i := len(chain) - 2; i >= 0; i-- {
		path += "/subsystems/" + chain[i]
	}
	return path, nil
}

func (r *RepositoryChangeResolver) deploymentPath(ctx context.Context, id string) (string, error) {
	var chain []string
	seen := map[string]bool{}
	for id != "" {
		if seen[id] || len(chain) >= 256 {
			return "", fmt.Errorf("cyclic or excessively deep deployment ownership")
		}
		seen[id] = true
		d, err := r.repos.Deployment.GetByID(ctx, id)
		if err != nil {
			return "", err
		}
		chain = append(chain, EncodeID(d.ID))
		id = ""
		if d.ParentDeploymentID != nil {
			id = *d.ParentDeploymentID
		}
	}
	if len(chain) == 0 {
		return "", fmt.Errorf("owning deployment is required")
	}
	path := "deployments/" + chain[len(chain)-1]
	for i := len(chain) - 2; i >= 0; i-- {
		path += "/subdeployments/" + chain[i]
	}
	return path, nil
}

func (r *RepositoryChangeResolver) controlPath(ctx context.Context, id string) (string, error) {
	c, err := r.repos.ControlStream.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if c.SystemID == nil {
		return "", fmt.Errorf("controlstream has no owning system")
	}
	p, err := r.SystemPath(ctx, *c.SystemID)
	if err != nil {
		return "", err
	}
	return p + "/controlstreams/" + EncodeID(id), nil
}

func (r *RepositoryChangeResolver) commandPath(ctx context.Context, id string) (string, error) {
	c, err := r.repos.Command.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	p, err := r.controlPath(ctx, c.ControlStreamID)
	if err != nil {
		return "", err
	}
	return p + "/commands/" + EncodeID(id), nil
}

func (r *RepositoryChangeResolver) topicPath(ctx context.Context, c Change) (string, error) {
	parts := splitPath(c.ParentPath)
	parent := ""
	if len(parts) > 0 {
		parent = parts[len(parts)-1]
	}
	id := EncodeID(c.ResourceID)
	switch c.ResourceType {
	case "system":
		return r.SystemPath(ctx, c.ResourceID)
	case "deployment":
		return r.deploymentPath(ctx, c.ResourceID)
	case "procedure":
		return "procedures/" + id, nil
	case "property":
		return "properties/" + id, nil
	case "datastream", "controlstream", "samplingfeature", "systemevent":
		p, err := r.SystemPath(ctx, parent)
		if err != nil {
			return "", err
		}
		collection := map[string]string{"datastream": "datastreams", "controlstream": "controlstreams", "samplingfeature": "samplingFeatures", "systemevent": "events"}[c.ResourceType]
		return p + "/" + collection + "/" + id, nil
	case "observation":
		d, err := r.repos.Datastream.GetByID(ctx, parent)
		if err != nil {
			return "", err
		}
		if d.SystemID == nil {
			return "", fmt.Errorf("datastream has no owning system")
		}
		p, err := r.SystemPath(ctx, *d.SystemID)
		if err != nil {
			return "", err
		}
		return p + "/datastreams/" + EncodeID(d.ID) + "/observations/" + id, nil
	case "command":
		return r.commandPath(ctx, c.ResourceID)
	case "commandstatus", "commandresult":
		p, err := r.commandPath(ctx, parent)
		if err != nil {
			return "", err
		}
		collection := "status"
		if c.ResourceType == "commandresult" {
			collection = "result"
		}
		return p + "/" + collection + "/" + id, nil
	}
	return "", fmt.Errorf("no MQTT ownership mapping for %s", c.ResourceType)
}

func collectionTopic(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i]
	}
	return ""
}
