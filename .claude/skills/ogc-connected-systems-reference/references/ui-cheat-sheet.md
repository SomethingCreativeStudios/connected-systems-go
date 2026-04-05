# UI Cheat Sheet

## Purpose

This guide translates the Connected Systems model into UI defaults.

It focuses on two questions:

- which UI pattern should each `IOComponent` variant use?
- how should the same renderer be reused across Part 1 design-time channels and Part 2 runtime payloads?

## Core Rule

Use one shared component vocabulary across both layers:

- Part 1 `inputs`, `outputs`, and `parameters`
- Part 2 datastream `schema.resultSchema`
- Part 2 datastream `schema.parametersSchema`
- Part 2 control-stream `schema.parametersSchema`
- Part 2 control-stream `schema.resultSchema`
- Part 2 control-stream `schema.feasibilityResultSchema`

Practical UI rule:

- classify the `IOComponent`
- pick a renderer from that classification
- reuse the same renderer for both schema authoring and runtime payload display

## UI Layers

Use this split in the product:

- schema authoring UI: define or inspect the Part 1 channel shape
- stream binding UI: choose `outputName` or `inputName`
- runtime value UI: render observations, commands, status results, and feasibility results from the same shape

Short version:

- Part 1 tells you what the channel is
- Part 2 tells you where live values flow through that channel

## Renderer Selection Order

For any schema node, use this order:

1. read `type`
2. decide scalar, range, record, vector, array, matrix, choice, or geometry
3. decide editable versus read-only based on context
4. decide single-value form, grouped form, table editor, map widget, or structured viewer

## Scalar Renderers

### Boolean

Default widget:

- toggle or checkbox

Use for:

- command forms
- configuration panels
- observation inspectors when boolean results are expected

Notes:

- prefer a switch when the meaning is operational
- prefer a checkbox when it is part of a larger settings form

### Count

Default widget:

- integer input with stepper

Use for:

- counters
- discrete actuator steps
- index-like parameters

Notes:

- respect integer-only validation
- show min or max constraints when available

### Quantity

Default widget:

- numeric input or numeric readout with unit adornment

Use for:

- temperatures
- distances
- angles
- speeds
- any measured real-valued channel

Notes:

- show `uom` next to the control, not buried in help text
- for read-only datastream values, use a compact metric tile or chart label when appropriate

### Time

Default widget:

- date-time input or timestamp viewer

Use for:

- schedule parameters
- execution times
- observed temporal values

Notes:

- display temporal reference context when present
- distinguish phenomenon time from result time in observation UIs

### Category

Default widget:

- select, combobox, segmented control, or tag display

Use for:

- enumerated modes
- controlled states
- coded classifications

Notes:

- if constraints enumerate values, render as select
- if values are open but code-based, render as combobox with code-space hint

### Text

Default widget:

- text input or multiline text area

Use for:

- notes
- identifiers entered as free text
- operator messages

Notes:

- multiline only when the definition implies narrative content
- otherwise prefer single-line input

## Range Renderers

### CountRange

Default widget:

- two integer inputs or min-max range pair

### QuantityRange

Default widget:

- dual numeric inputs or range slider with unit label

### TimeRange

Default widget:

- start and end date-time picker pair

### CategoryRange

Default widget:

- start and end coded value pair or constrained paired select

General rule:

- use paired controls when precision matters
- use a slider only when the range is small, numeric, and user-facing

## Aggregate Renderers

### DataRecord

Default widget:

- grouped form section or structured property panel

Best for:

- command forms
- observation detail views
- settings panes

Rendering rule:

- each `fields` item becomes one labeled child control or child display block

UI pattern:

- form card with one row per field for editing
- definition list or structured table for read-only display

### Vector

Default widget:

- grouped coordinate editor or compact coordinate table

Best for:

- position
- orientation
- velocity
- direction vectors

Rendering rule:

- render one control per coordinate in order
- show `referenceFrame` and `localFrame` visibly near the widget

UI pattern:

- 2D or 3D coordinate block
- optional mini spatial preview when the meaning is location-like

### DataArray

Default widget:

- table editor, grid viewer, or series viewer

Best for:

- repeated samples
- profiles
- trajectories
- row-oriented sensor output

Rendering rule:

- `elementType` defines the repeated row or repeated item structure
- if the element type is scalar, render a single-column list or series
- if the element type is a record, render a table with one column per record field

UI pattern:

- editable grid for commands or configuration
- read-only table or chart for observations

### Matrix

Default widget:

- matrix grid or specialized transform viewer

Best for:

- transforms
- calibration matrices
- covariance-style payloads

Rendering rule:

- preserve row or column structure visually
- show frame context next to the matrix

UI pattern:

- spreadsheet-style grid for editing
- formatted matrix viewer for read-only display

### DataChoice

Default widget:

- discriminated union selector plus branch-specific subform

Best for:

- alternate command payload families
- alternate result payload families
- polymorphic record content

Rendering rule:

- `choiceValue` drives the active branch
- once selected, render only the chosen branch from `items`

UI pattern:

- radio group, select, or tab switcher for the choice
- nested renderer for the selected branch

## Geometry Renderer

### Geometry

Default widget:

- map picker, geometry editor, or geometry viewer

Best for:

- points
- lines
- polygons
- spatial footprints

Rendering rule:

- use the map as the primary editor when geometry is user-authored
- show `srs` and geometry type visibly
- respect `constraint.geomTypes` when limiting allowed geometries

UI pattern:

- map drawing tool for editable geometry
- mini map preview for read-only geometry

## Context-Specific Defaults

### Part 1 Procedure Or System Editor

Use the UI to define channel structure.

Recommended behavior:

- show full schema structure
- allow nested editing of `DataRecord`, `Vector`, `DataArray`, `Matrix`, and `DataChoice`
- emphasize semantic metadata such as `definition`, `label`, `uom`, `codeSpace`, and constraints

### Datastream Creation UI

Use the UI to bind a runtime stream to one output.

Recommended behavior:

- pick the system first
- select `outputName` from the system output list
- prefill `schema.resultSchema` from the bound output definition
- optionally prefill `schema.parametersSchema` from related Part 1 parameters when observation parameters are supported
- show derived stream context links as read-only or hidden

Important implication:

- the UI should not ask users to redesign the schema from scratch if the system output already defines it

### ControlStream Creation UI

Use the UI to bind a runtime control channel to one input.

Recommended behavior:

- pick the system first
- select `inputName` from the system input list
- prefill `schema.parametersSchema` from the bound input definition
- prefill `schema.resultSchema` or feasibility result schema only when the command workflow publishes structured results
- show derived context links as read-only or hidden

### Observation UI

Use the UI to render one runtime result instance.

Recommended behavior:

- render `result` using the same renderer chosen from the parent datastream `resultSchema`
- render `parameters` using the same renderer chosen from the parent datastream `parametersSchema`
- keep `samplingFeature@id` as a focused context field, not the main payload area

### Command UI

Use the UI to render one actuation request instance.

Recommended behavior:

- render `parameters` using the same renderer chosen from the parent control-stream `parametersSchema`
- keep timing and status fields read-only and visually separate from the editable payload

### Feasibility Or Result UI

Use the UI to render response payloads from the same schema family.

Recommended behavior:

- reuse the same component renderer used for commands and observations
- switch to read-only presentation unless the workflow is explicitly draft or simulation editing

## Read-Only Versus Editable Rules

Use schema shape to choose renderer type, but use resource context to choose editability.

Editable in most create flows:

- command `parameters`
- observation `result` when manually entered
- stream `schema` during creation or controlled edits

Usually read-only in normal UIs:

- `system@link`
- derived `procedure@link`, `deployment@link`, `samplingFeature@link`, `featureOfInterest@link`
- stream property summaries
- IDs
- runtime status fields

## Recommended Visual Shortcuts

Use these condensed patterns for common dashboards:

- `Quantity`: metric card, sparkline tile, or unit-labeled numeric cell
- `Boolean`: status pill or on-off badge
- `Category`: chip or tag
- `Time`: timeline label or relative-time badge with detail on hover
- `DataRecord`: expandable structured card
- `Vector`: coordinate badge row or mini table
- `DataArray`: chart plus optional raw table
- `Geometry`: map preview

## UI Automation Rules

To automate form generation safely:

1. start from the bound Part 1 channel if available
2. fall back to the Part 2 stream schema when the stream has been narrowed
3. preserve semantic metadata like `definition`, `label`, `uom`, and `codeSpace`
4. render constraints as validation hints and widgets, not as hidden logic
5. keep server-derived context links out of the editable form surface

## Concrete Examples

These examples show the same structural idea in both layers:

- Part 1 channel definition in SensorML-style IOComponent form
- Part 2 runtime schema or payload using the same renderer family

### Quantity Example

Part 1 output definition:

```json
{
	"type": "Quantity",
	"definition": "https://example.com/def/property/airTemperature",
	"label": "Air Temperature",
	"uom": {
		"code": "Cel"
	}
}
```

Part 2 datastream schema reuse:

```json
{
	"obsFormat": "application/json",
	"resultSchema": {
		"type": "Quantity",
		"definition": "https://example.com/def/property/airTemperature",
		"label": "Air Temperature",
		"uom": {
			"code": "Cel"
		}
	}
}
```

Part 2 observation runtime payload:

```json
{
	"phenomenonTime": "2026-04-04T12:00:00Z",
	"result": 21.4
}
```

Recommended UI:

- Part 1: numeric schema row with unit chip and semantic definition
- Part 2 stream editor: prefilled numeric result schema with unit label
- Part 2 runtime viewer: metric tile, compact numeric cell, or line chart point

### DataRecord Example

Part 1 input definition:

```json
{
	"type": "DataRecord",
	"definition": "https://example.com/def/input/ptz",
	"label": "PTZ Command",
	"fields": [
		{
			"name": "pan",
			"type": "Quantity",
			"definition": "https://example.com/def/property/pan",
			"label": "Pan",
			"uom": {
				"code": "deg"
			}
		},
		{
			"name": "tilt",
			"type": "Quantity",
			"definition": "https://example.com/def/property/tilt",
			"label": "Tilt",
			"uom": {
				"code": "deg"
			}
		},
		{
			"name": "zoom",
			"type": "Count",
			"definition": "https://example.com/def/property/zoom",
			"label": "Zoom"
		}
	]
}
```

Part 2 control-stream schema reuse:

```json
{
	"commandFormat": "application/json",
	"parametersSchema": {
		"type": "DataRecord",
		"definition": "https://example.com/def/input/ptz",
		"label": "PTZ Command",
		"fields": [
			{
				"name": "pan",
				"type": "Quantity",
				"definition": "https://example.com/def/property/pan",
				"label": "Pan",
				"uom": {
					"code": "deg"
				}
			},
			{
				"name": "tilt",
				"type": "Quantity",
				"definition": "https://example.com/def/property/tilt",
				"label": "Tilt",
				"uom": {
					"code": "deg"
				}
			},
			{
				"name": "zoom",
				"type": "Count",
				"definition": "https://example.com/def/property/zoom",
				"label": "Zoom"
			}
		]
	}
}
```

Part 2 command runtime payload:

```json
{
	"parameters": {
		"pan": 30,
		"tilt": 10,
		"zoom": 4
	}
}
```

Recommended UI:

- Part 1: grouped schema editor with one row per field
- Part 2 stream editor: prefilled form schema, not a blank custom JSON editor
- Part 2 runtime form: command card with labeled child inputs for pan, tilt, and zoom

### DataArray Example

Part 1 output definition:

```json
{
	"type": "DataArray",
	"definition": "https://example.com/def/output/spectrum",
	"label": "Spectrum",
	"elementType": {
		"type": "DataRecord",
		"label": "Spectrum Sample",
		"fields": [
			{
				"name": "wavelength",
				"type": "Quantity",
				"definition": "https://example.com/def/property/wavelength",
				"label": "Wavelength",
				"uom": {
					"code": "nm"
				}
			},
			{
				"name": "intensity",
				"type": "Quantity",
				"definition": "https://example.com/def/property/intensity",
				"label": "Intensity",
				"uom": {
					"code": "1"
				}
			}
		]
	}
}
```

Part 2 datastream schema reuse:

```json
{
	"obsFormat": "application/json",
	"resultSchema": {
		"type": "DataArray",
		"definition": "https://example.com/def/output/spectrum",
		"label": "Spectrum",
		"elementType": {
			"type": "DataRecord",
			"label": "Spectrum Sample",
			"fields": [
				{
					"name": "wavelength",
					"type": "Quantity",
					"definition": "https://example.com/def/property/wavelength",
					"label": "Wavelength",
					"uom": {
						"code": "nm"
					}
				},
				{
					"name": "intensity",
					"type": "Quantity",
					"definition": "https://example.com/def/property/intensity",
					"label": "Intensity",
					"uom": {
						"code": "1"
					}
				}
			]
		}
	}
}
```

Part 2 observation runtime payload:

```json
{
	"result": [
		{
			"wavelength": 450,
			"intensity": 0.31
		},
		{
			"wavelength": 550,
			"intensity": 0.62
		},
		{
			"wavelength": 650,
			"intensity": 0.28
		}
	]
}
```

Recommended UI:

- Part 1: repeated-row schema editor showing the record columns
- Part 2 stream editor: preview table columns from `elementType`
- Part 2 runtime viewer: chart plus expandable raw table

### Geometry Example

Part 1 parameter or output definition:

```json
{
	"type": "Geometry",
	"definition": "https://example.com/def/property/footprint",
	"label": "Footprint",
	"srs": "http://www.opengis.net/def/crs/EPSG/0/4326",
	"constraint": {
		"geomTypes": [
			"Polygon"
		]
	}
}
```

Part 2 control-stream schema reuse:

```json
{
	"commandFormat": "application/json",
	"parametersSchema": {
		"type": "Geometry",
		"definition": "https://example.com/def/property/footprint",
		"label": "Footprint",
		"srs": "http://www.opengis.net/def/crs/EPSG/0/4326",
		"constraint": {
			"geomTypes": [
				"Polygon"
			]
		}
	}
}
```

Part 2 command runtime payload:

```json
{
	"parameters": {
		"type": "Polygon",
		"coordinates": [
			[
				[-122.1, 47.6],
				[-122.0, 47.6],
				[-122.0, 47.7],
				[-122.1, 47.7],
				[-122.1, 47.6]
			]
		]
	}
}
```

Recommended UI:

- Part 1: geometry schema row with CRS and allowed geometry types
- Part 2 stream editor: prefilled geometry schema, not a free-text JSON blob
- Part 2 runtime form: map drawing widget restricted to polygons

## Quick Matrix

| IO type | Primary UI | Secondary UI | Typical use |
| --- | --- | --- | --- |
| `Boolean` | toggle | status badge | on/off channels |
| `Count` | integer input | compact numeric cell | counts, steps |
| `Quantity` | numeric input with unit | metric tile | measured values |
| `Time` | date-time picker | timestamp display | schedules, time payloads |
| `Category` | select or combobox | tag | states, coded values |
| `Text` | text input | text block | notes, free text |
| `CountRange` | min-max pair | summary chip | count intervals |
| `QuantityRange` | dual numeric pair | range summary | operating ranges |
| `TimeRange` | start-end picker | timeline chip | windows, schedules |
| `CategoryRange` | paired coded inputs | range label | coded intervals |
| `DataRecord` | grouped form | structured card | composite payloads |
| `Vector` | coordinate editor | coordinate table | framed tuples |
| `DataArray` | grid or chart | raw table | repeated samples |
| `Matrix` | matrix grid | formatted viewer | transforms |
| `DataChoice` | branch selector plus subform | selected-branch viewer | union payloads |
| `Geometry` | map editor | map preview | spatial payloads |

## Fast Checklist

Before implementing a new form or viewer, ask:

- what `IOComponent` family is this?
- is this schema coming from Part 1, Part 2, or both?
- should the user edit the payload or just inspect it?
- can we prefill the stream schema from the bound system channel?
- are we hiding server-derived context fields and exposing only the true payload?