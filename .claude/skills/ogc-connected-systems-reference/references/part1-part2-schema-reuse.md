# Part 1 To Part 2 Schema Reuse

## Purpose

This reference makes one implementation-critical point explicit:

- the SensorML `IOComponent` family used for Part 1 `inputs`, `outputs`, and `parameters` is also the same schema family used by Part 2 stream schemas for `resultSchema`, `parametersSchema`, and related record payloads

That means the Part 1 descriptive channel model and the Part 2 runtime payload model are not just conceptually aligned. They are structurally aligned.

## Core Rule

Use this equivalence:

- Part 1 `output` IOComponent -> Part 2 datastream `schema.resultSchema`
- Part 1 `parameter` IOComponent -> Part 2 datastream `schema.parametersSchema` when observation parameters are needed
- Part 1 `input` IOComponent -> Part 2 control-stream `schema.parametersSchema`
- Part 1 output-like result payloads -> Part 2 control-stream `schema.resultSchema` or `schema.feasibilityResultSchema` when command processing returns structured data

Practical implication:

- a system or procedure channel definition can often be reused almost directly when creating a datastream or control stream schema

## Why The Reuse Is Real In This Repo

The local Part 2 bundled JSON schemas point their stream schema branches back into the same SWE component family.

High-signal examples from the repo:

- datastream JSON `schema.resultSchema` ultimately points at the SWE `recordSchema` branch used for scalar, record, vector, array, matrix, choice, and geometry components
- datastream JSON `schema.parametersSchema` reuses that same branch
- control-stream JSON `schema.parametersSchema`, `schema.resultSchema`, and `schema.feasibilityResultSchema` also reuse that same branch

So when you recognize an IO component from Part 1, you are usually looking at the same structural vocabulary that Part 2 stream schemas expect.

## Mapping Chain

Use this chain when reading or generating resources:

- `Procedure` defines reusable `inputs`, `outputs`, and `parameters`
- `System` instantiates or narrows those channels for a concrete asset
- `Datastream.outputName` selects one concrete system output
- `ControlStream.inputName` selects one concrete system input
- the Part 2 stream `schema` should then reuse the same IOComponent shape as the selected Part 1 channel

Short version:

- procedure defines the reusable shape
- system provides the concrete bind point
- stream schema operationalizes that same shape at runtime

## Output Side Reuse

Output-side chain:

- procedure output
- system output
- `Datastream.outputName`
- datastream `schema.resultSchema`
- observation `result`

Meaning:

- if a system output is a `Quantity`, the datastream result schema can usually be the same `Quantity` shape
- if a system output is a `DataRecord`, the datastream result schema can usually be the same `DataRecord` shape
- if a system output is a `Vector`, `DataArray`, `Matrix`, `DataChoice`, or `Geometry`, the datastream can usually expose that same structure directly

This is the cleanest way to preserve semantic and structural continuity from Part 1 into Part 2.

## Input Side Reuse

Input-side chain:

- procedure input
- system input
- `ControlStream.inputName`
- control-stream `schema.parametersSchema`
- command `parameters`

Meaning:

- if a system input is a `Category`, `Quantity`, or `DataRecord`, the control-stream parameters schema can usually be that same shape
- command payloads then become direct runtime instances of the same input model described in Part 1

## Parameter Reuse

Part 1 `parameters` also map cleanly into Part 2 when a runtime item needs explicit parameter metadata.

Useful patterns:

- procedure or system parameter component -> datastream `schema.parametersSchema` for observation parameter blocks
- procedure or system parameter component -> nested command parameter records when the actuation model includes tuning or execution modifiers

This is especially useful when the measured or controlled value is simple, but the runtime call still needs structured context.

## UI Automation Use

This reuse is useful for UI automation because it lets the UI generate forms and viewers from one shared component vocabulary.

Practical automation model:

- read the Part 1 system or procedure channel definition
- identify the selected output or input by `outputName` or `inputName`
- reuse the same IOComponent renderer to generate:
  - datastream result viewers
  - observation result forms or inspectors
  - control-stream command forms
  - feasibility result viewers

That means one parser can usually drive both:

- design-time forms from Part 1 SensorML
- runtime forms from Part 2 stream schemas

## Safe Reuse Rules

Treat direct reuse as the default, but not an unconditional copy-paste rule.

Safe default:

- keep the same component family and field structure
- preserve `definition`, `label`, units, code spaces, and nested record structure

Allow narrowing when needed:

- a concrete system may narrow a reusable procedure channel
- a stream may omit optional branches not used in practice
- a stream may add outer encoding details such as `obsFormat`, `commandFormat`, or `resultLink`

Do not break the semantic contract:

- do not bind a datastream to one output and then use an unrelated schema shape
- do not bind a control stream to one input and then publish incompatible command parameters

## Quick Examples

### Scalar Output To Datastream

- Part 1 output: `Quantity` with temperature semantics and `uom`
- Part 2 datastream `resultSchema`: same `Quantity` shape
- Observation `result`: one runtime quantity value matching that shape

### Record Input To ControlStream

- Part 1 input: `DataRecord` with fields like pan, tilt, zoom
- Part 2 control-stream `parametersSchema`: same `DataRecord` shape
- Command `parameters`: one runtime PTZ request matching that shape

### Array Output To Datastream

- Part 1 output: `DataArray` of repeated samples
- Part 2 datastream `resultSchema`: same `DataArray` shape
- Observation `result`: one runtime array block matching that shape

## Validation Checklist

When using Part 1 definitions to automate Part 2 schema creation, ask:

- does `outputName` or `inputName` point to a real system channel?
- does that channel still align with the linked procedure channel?
- does the chosen Part 2 schema stay in the same IOComponent family?
- are units, code spaces, labels, and nested record fields preserved?
- did we add only runtime wrapper details such as encoding, not semantic drift?