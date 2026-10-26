package pubsub

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
)

// Capability is the single catalog of offered Part 3 paths and representations.
// SystemPath and deploymentPath span complete ownership chains; each ID is one level.
type Capability struct {
	Name, ResourceType, EventPath, CollectionPath, ParentCollection string
	MediaTypes                                                      []string
	Batch                                                           bool
	Operations                                                      []Operation
	ClientCreates, ServerPublishes                                  bool
}

var capabilities = []Capability{
	offer("system", "system", "systems/{id}", "systems", "", []string{mediatype.GeoJSON, mediatype.SensorML}, false),
	offer("subsystem", "system", "{systemPath}/subsystems/{id}", "{systemPath}/subsystems", "systems", []string{mediatype.GeoJSON, mediatype.SensorML}, false),
	offer("datastream", "datastream", "{systemPath}/datastreams/{id}", "{systemPath}/datastreams", "systems", []string{mediatype.JSON}, false),
	offer("controlstream", "controlstream", "{systemPath}/controlstreams/{id}", "{systemPath}/controlstreams", "systems", []string{mediatype.JSON}, false),
	offer("samplingfeature", "samplingfeature", "{systemPath}/samplingFeatures/{id}", "{systemPath}/samplingFeatures", "systems", nil, false),
	offer("systemevent", "systemevent", "{systemPath}/events/{id}", "{systemPath}/events", "systems", []string{mediatype.JSON}, false),
	offer("observation", "observation", "{systemPath}/datastreams/{datastreamId}/observations/{id}", "{systemPath}/datastreams/{datastreamId}/observations", "datastreams", []string{mediatype.JSON}, true),
	offer("command", "command", "{systemPath}/controlstreams/{controlstreamId}/commands/{id}", "{systemPath}/controlstreams/{controlstreamId}/commands", "controlstreams", []string{mediatype.JSON}, true),
	offer("commandstatus", "commandstatus", "{systemPath}/controlstreams/{controlstreamId}/commands/{commandId}/status/{id}", "{systemPath}/controlstreams/{controlstreamId}/commands/{commandId}/status", "commands", []string{mediatype.JSON}, false),
	offer("commandresult", "commandresult", "{systemPath}/controlstreams/{controlstreamId}/commands/{commandId}/result/{id}", "{systemPath}/controlstreams/{controlstreamId}/commands/{commandId}/result", "commands", []string{mediatype.JSON}, false),
	offer("deployment", "deployment", "deployments/{id}", "deployments", "", nil, false),
	offer("subdeployment", "deployment", "{deploymentPath}/subdeployments/{id}", "{deploymentPath}/subdeployments", "deployments", nil, false),
	offer("procedure", "procedure", "procedures/{id}", "procedures", "", nil, false),
	offer("property", "property", "properties/{id}", "properties", "", nil, false),
}

func offer(name, kind, event, collection, parent string, formats []string, batch bool) Capability {
	return Capability{Name: name, ResourceType: kind, EventPath: event, CollectionPath: collection, ParentCollection: parent, MediaTypes: formats, Batch: batch,
		Operations: []Operation{OperationCreate, OperationUpdate, OperationDelete}, ClientCreates: len(formats) > 0, ServerPublishes: len(formats) > 0}
}

func CapabilityForResource(kind string) (Capability, bool) {
	for _, c := range capabilities {
		if c.ResourceType == kind {
			return c, true
		}
	}
	return Capability{}, false
}

// CreatePath names the canonical HTTP collection. parent may be a URI-template parameter.
func (c Capability) CreatePath(parent string) string {
	if c.ParentCollection == "" {
		return "/" + c.CollectionPath
	}
	parts := strings.Split(c.CollectionPath, "/")
	return "/" + c.ParentCollection + "/" + parent + "/" + parts[len(parts)-1]
}

func Capabilities() []Capability {
	out := slices.Clone(capabilities)
	for i := range out {
		out[i].MediaTypes = slices.Clone(out[i].MediaTypes)
		out[i].Operations = slices.Clone(out[i].Operations)
	}
	return out
}

func EventTokens(batchEnabled, batchOnly bool) []string {
	var tokens []string
	for _, c := range capabilities {
		if c.Batch != batchOnly && (batchOnly || batchEnabled) {
			continue
		}
		for _, operation := range c.Operations {
			token := c.ResourceType + "." + string(operation)
			if !slices.Contains(tokens, token) {
				tokens = append(tokens, token)
			}
		}
	}
	return tokens
}

func DataTypes() []string {
	var types []string
	for _, c := range capabilities {
		if len(c.MediaTypes) > 0 && !slices.Contains(types, c.ResourceType) {
			types = append(types, c.ResourceType)
		}
	}
	return types
}

// EncodeID encodes bytes, including percent itself, to avoid aliases and path injection.
func EncodeID(id string) string {
	const hex = "0123456789ABCDEF"
	var out strings.Builder
	for i := 0; i < len(id); i++ {
		b := id[i]
		if b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || strings.ContainsRune("-._~", rune(b)) {
			out.WriteByte(b)
		} else {
			out.WriteByte('%')
			out.WriteByte(hex[b>>4])
			out.WriteByte(hex[b&15])
		}
	}
	return out.String()
}

func DecodeID(level string) (string, error) {
	id, err := url.PathUnescape(level)
	if err != nil || id == "" || !utf8.ValidString(id) || strings.ContainsRune(id, 0) || EncodeID(id) != level {
		return "", fmt.Errorf("invalid or noncanonical MQTT identifier %q", level)
	}
	return id, nil
}

func FormatName(mediaType string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(strings.SplitN(mediaType, ";", 2)[0], "application/"), "+", "-"))
}

func DataTopic(node, collectionPath, mediaType, defaultType string) string {
	topic := node + "/" + strings.Trim(collectionPath, "/") + ":data"
	if mediaType != defaultType {
		topic += "/" + FormatName(mediaType)
	}
	return topic
}

type Target struct {
	Capability                                                    Capability
	CollectionTopicPath, CanonicalCollection, ParentID, MediaType string
}

var (
	placeholder       = regexp.MustCompile(`\{[^}]+\}`)
	quotedPlaceholder = regexp.MustCompile(`\\\{[^}]+\\\}`)
	// Authorization runs several times per MQTT delivery, so every catalog
	// template is compiled once rather than on each match.
	templatePatterns = func() map[string]*regexp.Regexp {
		patterns := make(map[string]*regexp.Regexp)
		for _, c := range capabilities {
			patterns[c.EventPath] = compileTemplate(c.EventPath)
			patterns[c.CollectionPath] = compileTemplate(c.CollectionPath)
		}
		return patterns
	}()
)

func compileTemplate(template string) *regexp.Regexp {
	pattern := regexp.QuoteMeta(template)
	pattern = strings.ReplaceAll(pattern, regexp.QuoteMeta("{systemPath}"), `systems/[^/]+(?:/subsystems/[^/]+)*`)
	pattern = strings.ReplaceAll(pattern, regexp.QuoteMeta("{deploymentPath}"), `deployments/[^/]+(?:/subdeployments/[^/]+)*`)
	pattern = quotedPlaceholder.ReplaceAllString(pattern, `[^/]+`)
	return regexp.MustCompile("^" + pattern + "$")
}

func matchPath(template, path string) bool {
	pattern, ok := templatePatterns[template]
	if !ok {
		pattern = compileTemplate(template)
	}
	if !pattern.MatchString(path) {
		return false
	}
	parts := strings.Split(path, "/")
	for i := 1; i < len(parts); i += 2 {
		if _, err := DecodeID(parts[i]); err != nil {
			return false
		}
	}
	return true
}

func ParseDataTopic(node, topic string) (Target, error) {
	var target Target
	path, ok := strings.CutPrefix(topic, node+"/")
	if !ok {
		return target, fmt.Errorf("topic does not belong to node %q", node)
	}
	collection, format, hasFormat := strings.Cut(path, ":data")
	if collection == "" || (hasFormat && format != "" && (!strings.HasPrefix(format, "/") || strings.Count(format, "/") != 1)) {
		return target, fmt.Errorf("invalid Resource Data topic")
	}
	if !hasFormat {
		return target, fmt.Errorf("not a Resource Data topic")
	}
	for _, c := range capabilities {
		if len(c.MediaTypes) == 0 || !matchPath(c.CollectionPath, collection) {
			continue
		}
		media := c.MediaTypes[0]
		if format != "" {
			media = ""
			for _, m := range c.MediaTypes[1:] {
				if "/"+FormatName(m) == format {
					media = m
				}
			}
			if media == "" {
				return target, fmt.Errorf("unsupported format subtopic %q", format)
			}
		}
		parts := strings.Split(collection, "/")
		canonical := "/" + parts[len(parts)-1]
		parent := ""
		if c.ParentCollection != "" {
			parent, _ = DecodeID(parts[len(parts)-2])
			canonical = c.CreatePath(url.PathEscape(parent))
		}
		return Target{c, collection, canonical, parent, media}, nil
	}
	return target, fmt.Errorf("topic is not an offered Resource Data collection")
}

func TopicOffered(node, topic string, cfg config.PubSubConfig) bool {
	if strings.Contains(topic, ":data") {
		_, err := ParseDataTopic(node, topic)
		return cfg.ResourceData.Enabled && err == nil
	}
	path, ok := strings.CutPrefix(topic, node+"/")
	if !ok {
		return false
	}
	for _, c := range capabilities {
		if cfg.ResourceEvents.Enabled && (!c.Batch || !cfg.BatchResourceEvents.Enabled) && matchPath(c.EventPath, path) {
			return true
		}
		if cfg.ResourceEvents.Enabled && cfg.BatchResourceEvents.Enabled && c.Batch && matchPath(c.CollectionPath, path) {
			return true
		}
	}
	return false
}

// FilterOffered checks a filter against templates, not today's resource inventory.
func FilterOffered(node, filter string, cfg config.PubSubConfig) bool {
	parts := strings.Split(filter, "/")
	for i, p := range parts {
		if strings.ContainsAny(p, "+#") && (p != "+" && (p != "#" || i != len(parts)-1)) {
			return false
		}
	}
	if !strings.ContainsAny(filter, "+#") {
		return TopicOffered(node, filter, cfg)
	}
	for _, c := range capabilities {
		var templates []string
		if cfg.ResourceEvents.Enabled && (!c.Batch || !cfg.BatchResourceEvents.Enabled) {
			templates = append(templates, c.EventPath)
		}
		if cfg.ResourceEvents.Enabled && cfg.BatchResourceEvents.Enabled && c.Batch {
			templates = append(templates, c.CollectionPath)
		}
		if cfg.ResourceData.Enabled && len(c.MediaTypes) > 0 {
			templates = append(templates, c.CollectionPath+":data")
			for _, m := range c.MediaTypes[1:] {
				templates = append(templates, c.CollectionPath+":data/"+FormatName(m))
			}
		}
		for _, template := range templates {
			for depth := 0; depth <= len(parts); depth++ {
				path := strings.ReplaceAll(template, "{systemPath}", "systems/{id}"+strings.Repeat("/subsystems/{id}", depth))
				path = strings.ReplaceAll(path, "{deploymentPath}", "deployments/{id}"+strings.Repeat("/subdeployments/{id}", depth))
				candidate := strings.Split(node+"/"+path, "/")
				for i, p := range candidate {
					if placeholder.MatchString(p) {
						value := "resource"
						if i < len(parts) && parts[i] != "+" && parts[i] != "#" {
							value = parts[i]
						}
						candidate[i] = placeholder.ReplaceAllString(p, value)
					}
				}
				if topicMatches(strings.Join(candidate, "/"), filter) && TopicOffered(node, strings.Join(candidate, "/"), cfg) {
					return true
				}
			}
		}
	}
	return false
}

func topicMatches(topic, filter string) bool {
	if strings.HasPrefix(topic, "$") && (strings.HasPrefix(filter, "#") || strings.HasPrefix(filter, "+")) {
		return false
	}
	t, f := strings.Split(topic, "/"), strings.Split(filter, "/")
	for i, p := range f {
		if p == "#" {
			return i == len(f)-1
		}
		if i >= len(t) || p != "+" && p != t[i] {
			return false
		}
	}
	return len(t) == len(f)
}

// ConformanceClasses derives enabled draft classes from the offered catalog.
// The draft ATS has no final identifiers; these URIs remain provisional.
func ConformanceClasses(cfg config.PubSubConfig) []string {
	const base = "http://www.opengis.net/spec/ogcapi-connectedsystems-3/1.0/conf/"
	var events, batch, data, formats bool
	for _, c := range capabilities {
		events = events || cfg.ResourceEvents.Enabled && len(c.Operations) > 0
		batch = batch || cfg.ResourceEvents.Enabled && cfg.BatchResourceEvents.Enabled && c.Batch
		data = data || cfg.ResourceData.Enabled && c.ServerPublishes && len(c.MediaTypes) > 0
		formats = formats || cfg.ResourceData.Enabled && len(c.MediaTypes) > 1
	}
	var classes []string
	if events {
		classes = append(classes, base+"resource-events")
	}
	if batch {
		classes = append(classes, base+"resource-events/batch-events")
	}
	if data {
		classes = append(classes, base+"resource-data-messages")
	}
	if formats {
		classes = append(classes, base+"resource-data-messages/content-negotiation")
	}
	if len(classes) > 0 {
		classes = append(classes, base+"mqtt")
	}
	return classes
}
