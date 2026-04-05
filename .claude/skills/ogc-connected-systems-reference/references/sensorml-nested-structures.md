# SensorML Nested Structures

## Purpose

This guide explains the high-noise nested blocks that appear on SensorML `Procedure` and `System` resources:

- `configuration`
- `modes`
- `capabilities`
- `characteristics`

These fields are easy to misread because they mix reusable metadata, constrained runtime settings, and grouped IO-style data components.

## Mental Model

Use this split first:

- `characteristics`: what the thing is
- `capabilities`: what the thing can do
- `configuration`: what has been set or constrained for this instance or mode
- `modes`: named presets that apply a configuration bundle

Practical rule:

- `characteristics` and `capabilities` are grouped descriptive inventories
- `configuration` and `modes` are grouped setting mechanisms

## Characteristics

Schema shape:

- top-level field: `characteristics`
- value: array of `CharacteristicList`
- each list requires `characteristics`

Important fields on each `CharacteristicList`:

- `definition`
- `conditions`
- `characteristics`

Interpretation:

- a `CharacteristicList` is a named group of mostly intrinsic properties
- `conditions` tells you when that group applies
- each item inside `characteristics` uses the same capability-item structure as the capabilities model, so the payload is still made of IO-style components

Read it as:

- group semantics first via `definition`
- applicability second via `conditions`
- actual per-property entries inside `characteristics`

Typical use:

- physical dimensions
- manufacturer-stated tolerances
- fixed identifiers or classifying properties
- environmental or operating assumptions attached to those properties

## Capabilities

Schema shape:

- top-level field: `capabilities`
- value: array of `CapabilityList`
- each list requires `capabilities`

Important fields on each `CapabilityList`:

- `definition`
- `conditions`
- `capabilities`

Important detail from the local bundled schema:

- each capability entry is not every possible IO type
- the allowed items are a narrower subset pulled from the IO component family

Interpretation:

- a `CapabilityList` groups things the process or system is able to provide, achieve, or support
- `conditions` narrows when the capability claim is valid
- entries are still data-component objects, so units, ranges, constraints, and labels matter

Read it as:

- what capability category this group describes
- under what conditions it applies
- what the reported numeric, categorical, or structured capability values are

Typical use:

- measurement range
- operating envelope
- throughput limits
- timing limits
- resolution or accuracy claims

## Characteristics Versus Capabilities

Use this distinction when the schema content looks ambiguous:

- if the value describes an inherent or identifying property, put it under `characteristics`
- if the value describes supported performance, range, or operating behavior, put it under `capabilities`

Short version:

- `characteristics` answers what it is like
- `capabilities` answers what it can do

## Configuration

Schema shape:

- top-level field: `configuration`
- value: object with several parallel setting arrays

Important child arrays in the local schema:

- `setValues`
- `setArrayValues`
- `setModes`
- `setConstraints`
- `setStatus`

All of these use a `ref` field that points into process structure using connection-style paths.

Those refs follow the same path vocabulary used by aggregate-process connections:

- `components/...`
- `inputs/...`
- `outputs/...`
- `parameters/...`
- `modes/...`

### setValues

Shape:

- array of `{ ref, value }`
- `value` is scalar-like in the local schema: number or string

Use when:

- setting a single parameter or component value
- overriding a base process default

### setArrayValues

Shape:

- array of `{ ref, value }`
- `value` is an array

Use when:

- the target component is array-valued
- a record needs repeated values rather than a single scalar

### setModes

Shape:

- array of `{ ref, value }`
- `value` is a string naming the selected mode

Use when:

- one configuration block activates another named preset rather than directly setting raw values

### setConstraints

Shape:

- array of constraint objects that all include `ref`
- the allowed constraint bodies are a union

Important local variants:

- token-style or existing SWE constraint reuse through the referenced IO schemas
- `AllowedValues`
- `AllowedTimes`

Interpretation:

- this does not just set a value
- it narrows the allowed domain of values for the referenced component

Use when:

- an inherited process model is too broad and the instance needs tighter bounds
- a mode narrows permissible time or numeric ranges

### setStatus

Shape:

- array of `{ ref, value }`
- `value` is `enabled` or `disabled`

Use when:

- switching a referenced component on or off without removing it structurally

## Modes

Schema shape:

- top-level field: `modes`
- value: array
- each mode item reuses the common descriptive process shell and adds its own `configuration`

Important implication:

- a mode is not just a string token
- it is a named object that can carry identifiers, labels, classifiers, documents, and a nested configuration block

Read it as:

- mode metadata defines what the preset is
- mode `configuration` defines what gets applied when that preset is selected

Practical rule:

- use `modes` to define reusable presets
- use top-level `configuration` to define the actual applied restrictions or settings for this concrete resource

## Configuration Versus Modes

Use this split:

- `configuration` is the current or inherited settings payload
- `modes` is the catalog of named presets

Then:

- `configuration.setModes` selects from that catalog
- each mode can itself carry a `configuration` block that says what the preset means

## Parsing Order For Large Objects

When a process document is large, parse these blocks in this order:

1. read `type`, `label`, and `uniqueId`
2. read `inputs`, `outputs`, and `parameters` to understand the interface surface
3. read `characteristics` for intrinsic grouped properties
4. read `capabilities` for grouped performance or operating claims
5. read top-level `configuration` for actual constrained settings
6. read `modes` only after configuration, because modes are presets layered on top of the same configuration model

## Common Misreads To Avoid

- Do not treat `modes` as plain enumerated strings; each mode is a structured object.
- Do not treat `configuration` as free-form JSON; it is a narrow schema built around `ref` plus typed setting arrays.
- Do not assume `characteristics` and `capabilities` are separate structural families; both ultimately carry IO-style component entries.
- Do not read `setConstraints` as current values; it defines permitted domains, not selected values.

## Connected Systems Relevance

This nested structure matters because Part 1 SensorML descriptions often explain the same domain semantics that Part 2 streams later operationalize.

In practice:

- `outputs` plus their configuration context often explain what becomes a `Datastream`
- `inputs` plus configuration and modes often explain what becomes a `ControlStream`
- `capabilities` and `characteristics` often explain why observed or controlled properties exist and what constraints they should respect