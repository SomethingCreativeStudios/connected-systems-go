# UI Anti-Patterns

## Purpose

This note captures the easiest UI mistakes to make when implementing Connected Systems forms and viewers.

The goal is to prevent the UI from fighting the model.

## Anti-Pattern 1: Editing Server-Derived Context Fields

Bad pattern:

- exposing editable controls for `system@link`, derived `procedure@link`, `deployment@link`, `samplingFeature@link`, `featureOfInterest@link`, stream property summaries, or server-issued IDs

Why it is wrong:

- these are structural, derived, or operational fields
- letting the user edit them makes the UI imply ownership the API model does not actually grant

Better pattern:

- show them as read-only metadata or hide them when they add no value
- let the user choose `outputName`, `inputName`, and other true client-owned fields instead

## Anti-Pattern 2: Rebuilding Stream Schemas From Scratch

Bad pattern:

- presenting an empty JSON editor for datastream or control-stream schema creation when the system channel already defines the shape

Why it is wrong:

- it duplicates work
- it increases drift between Part 1 channel definitions and Part 2 runtime schemas

Better pattern:

- bind to the system first
- choose `outputName` or `inputName`
- prefill the stream schema from the corresponding Part 1 IOComponent definition

## Anti-Pattern 3: Flattening `DataChoice` Into One Big Form

Bad pattern:

- merging all `DataChoice.items` fields into a single flat editor

Why it is wrong:

- `DataChoice` is a tagged union, not a record
- flattening it removes the active-branch semantics and creates impossible payloads

Better pattern:

- render a branch selector from `choiceValue`
- show only the currently selected branch form or viewer

## Anti-Pattern 4: Hiding Units, Code Spaces, And Frame Context

Bad pattern:

- rendering numbers without units
- rendering categories without `codeSpace` cues
- rendering vectors or matrices without `referenceFrame` or `localFrame`

Why it is wrong:

- the payload may still be structurally valid, but the UI loses the semantics users need to interpret it correctly

Better pattern:

- keep `uom`, `codeSpace`, `referenceFrame`, `localFrame`, and CRS context close to the value widget or display widget

## Anti-Pattern 5: Treating Constraints As Invisible Validation Only

Bad pattern:

- enforcing allowed values or geometry restrictions silently without reflecting them in the control

Why it is wrong:

- users cannot understand what values are expected
- the form feels arbitrary when validation fails

Better pattern:

- turn constraints into visible widget choices, ranges, hints, and restricted geometry tools

## Anti-Pattern 6: Rendering All Arrays The Same Way

Bad pattern:

- treating every `DataArray` as a generic JSON blob or plain textarea

Why it is wrong:

- scalar arrays, record arrays, and chartable series have very different interaction needs

Better pattern:

- inspect `elementType`
- render scalar arrays as series or lists
- render record arrays as tables
- add chart views when the semantics are time series or profiles

## Anti-Pattern 7: Mixing Editable Payload And Runtime Status

Bad pattern:

- placing command status, issue time, sender, and result publication fields inside the same editable block as command `parameters`

Why it is wrong:

- it mixes request authoring with server-side execution state

Better pattern:

- keep editable payloads in one form section
- keep status, timing, and result metadata in a separate read-only status panel

## Anti-Pattern 8: Losing The Part 1 To Part 2 Bind Point

Bad pattern:

- rendering a datastream or control-stream as if it were an independent schema root with no visible relation to the selected system channel

Why it is wrong:

- users lose the mental link between procedure or system design and runtime streams
- UI automation becomes harder because the source channel is obscured

Better pattern:

- visibly show the selected `outputName` or `inputName`
- optionally show the matched procedure or system channel label near the stream schema

## Anti-Pattern 9: Overusing Raw JSON Editors

Bad pattern:

- defaulting to raw JSON editing for every nested structure, even for common scalar, record, vector, and geometry types

Why it is wrong:

- it exposes implementation detail instead of domain semantics
- it makes validation and discoverability much worse

Better pattern:

- use structured editors by default
- keep raw JSON as an advanced fallback, not the primary workflow

## Anti-Pattern 10: Treating Observation Or Command Payloads As Self-Describing

Bad pattern:

- rendering runtime payloads without consulting the parent datastream or control-stream schema

Why it is wrong:

- the payload shape is defined by the parent stream contract
- the runtime item alone may not carry enough display metadata

Better pattern:

- always load the parent stream schema when rendering `result`, `parameters`, or result payloads
- let the parent schema drive the renderer and validation

## Fast Review Checklist

Before shipping a form or viewer, ask:

- are we editing only client-owned fields?
- are stream schemas prefilling from the bound system channel?
- are `DataChoice` branches rendered as alternatives rather than flattened fields?
- are units, code spaces, frames, and CRS visible?
- are constraints visible in the widget design?
- are runtime status fields separated from editable payloads?