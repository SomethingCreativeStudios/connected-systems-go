package json_formatters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// CommandJSONFeature is the application/json representation of a command.
// Its fields are listed explicitly so the API shape does not follow the
// database model.
type CommandJSONFeature struct {
	ID                string                   `json:"id"`
	ControlStreamID   string                   `json:"controlstream@id"`
	SamplingFeatureID *string                  `json:"samplingFeature@id,omitempty"`
	ProcedureLink     *common_shared.Link      `json:"procedure@link,omitempty"`
	IssueTime         *time.Time               `json:"issueTime"`
	ExecutionTime     *common_shared.TimeRange `json:"executionTime,omitempty"`
	Sender            string                   `json:"sender,omitempty"`
	CurrentStatus     domains.CommandStatus    `json:"currentStatus,omitempty"`
	Parameters        json.RawMessage          `json:"parameters,omitempty"`
}

// toDomain keeps the fields a client may write. issueTime is honored on
// create (the repository ignores it on update); id, controlstream@id,
// sender, currentStatus and executionTime are server-owned, the last two
// set only by status reports.
func (w CommandJSONFeature) toDomain() *domains.Command {
	return &domains.Command{
		SamplingFeatureID: w.SamplingFeatureID,
		ProcedureLink:     w.ProcedureLink,
		IssueTime:         w.IssueTime,
		Parameters:        w.Parameters,
	}
}

// CommandJSONFormatter handles command JSON serialization/deserialization.
type CommandJSONFormatter struct {
	links *formaters.LinkResolver
}

var _ formaters.Formatter[CommandJSONFeature, *domains.Command] = (*CommandJSONFormatter)(nil)

func NewCommandJSONFormatter(links *formaters.LinkResolver) *CommandJSONFormatter {
	return &CommandJSONFormatter{links: links}
}

func (f *CommandJSONFormatter) ContentType() string {
	return mediatype.JSON
}

func (f *CommandJSONFormatter) Serialize(ctx context.Context, cmd *domains.Command) (CommandJSONFeature, error) {
	return formaters.SerializeOne(ctx, cmd, f.SerializeAll)
}

func (f *CommandJSONFormatter) SerializeAll(ctx context.Context, commands []*domains.Command) ([]CommandJSONFeature, error) {
	commands = slices.DeleteFunc(slices.Clone(commands), func(cmd *domains.Command) bool { return cmd == nil })

	var procedureIDs []string
	for _, cmd := range commands {
		if cmd.ProcedureLink != nil {
			procedureIDs = appendID(procedureIDs, cmd.ProcedureLink.GetId("procedures"))
		}
	}
	cache, err := f.links.Lookup(ctx, formaters.References{Procedures: procedureIDs})
	if err != nil {
		return nil, err
	}

	items := make([]CommandJSONFeature, 0, len(commands))
	for _, cmd := range commands {
		procedure := inlineLink(f.links, cmd.ProcedureLink, mediatype.SensorML)
		describeProcedure(procedure, cache)
		items = append(items, CommandJSONFeature{
			ID:                cmd.ID,
			ControlStreamID:   cmd.ControlStreamID,
			SamplingFeatureID: cmd.SamplingFeatureID,
			ProcedureLink:     procedure,
			IssueTime:         cmd.IssueTime,
			ExecutionTime:     cmd.ExecutionTime,
			Sender:            cmd.Sender,
			CurrentStatus:     cmd.CurrentStatus,
			Parameters:        cmd.Parameters,
		})
	}
	return items, nil
}

func (f *CommandJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Command, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	wire, err := common_shared.DecodeWithFieldErrors[CommandJSONFeature](raw)
	if err != nil {
		return nil, err
	}
	// Required root fields per OGC CS Part 2 command schema
	if len(wire.Parameters) == 0 {
		return nil, fmt.Errorf("parameters is required")
	}
	return wire.toDomain(), nil
}
