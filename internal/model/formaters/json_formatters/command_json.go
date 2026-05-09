package json_formatters

import (
	"context"
	"fmt"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
)

// CommandJSONFeature is the wire-format representation of a command.
type CommandJSONFeature struct {
	domains.Command
}

// CommandJSONFormatter handles command JSON serialization/deserialization.
type CommandJSONFormatter struct {
	formaters.Formatter[CommandJSONFeature, *domains.Command]
}

func NewCommandJSONFormatter() *CommandJSONFormatter {
	return &CommandJSONFormatter{}
}

func (f *CommandJSONFormatter) ContentType() string {
	return JSONContentType
}

func (f *CommandJSONFormatter) Serialize(ctx context.Context, cmd *domains.Command) (CommandJSONFeature, error) {
	if cmd == nil {
		return CommandJSONFeature{}, fmt.Errorf("command cannot be nil")
	}
	out := CommandJSONFeature{Command: *cmd}
	if out.ProcedureLink != nil && out.ProcedureLink.Href != "" {
		out.ProcedureLink.Href = formaters.ToFunctionalAssociationHref(out.ProcedureLink.Href)
	}
	return out, nil
}

func (f *CommandJSONFormatter) SerializeAll(ctx context.Context, commands []*domains.Command) ([]CommandJSONFeature, error) {
	if len(commands) == 0 {
		return []CommandJSONFeature{}, nil
	}

	items := make([]CommandJSONFeature, 0, len(commands))
	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		out := CommandJSONFeature{Command: *cmd}
		if out.ProcedureLink != nil && out.ProcedureLink.Href != "" {
			out.ProcedureLink.Href = formaters.ToFunctionalAssociationHref(out.ProcedureLink.Href)
		}
		items = append(items, out)
	}
	return items, nil
}

func (f *CommandJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Command, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	wire, err := common_shared.DecodeWithFieldErrors[domains.Command](raw)
	if err != nil {
		return nil, err
	}
	return &wire, nil
}
