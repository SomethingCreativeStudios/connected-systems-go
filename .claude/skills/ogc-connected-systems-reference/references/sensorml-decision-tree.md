# SensorML Decision Tree

## Purpose

This is the fast classifier for the two biggest SensorML unions used in this repo:

- `AbstractProcess`
- `IOComponentChoice`

Use it when you need to identify a large object quickly before reading every field.

## Process Decision Tree

Start with `type`.

### If `type` is `SimpleProcess`

Read it as:

- logical leaf process

Expect:

- `method`
- no `components`

### If `type` is `AggregateProcess`

Read it as:

- logical composite process

Expect:

- `components`
- `connections`
- no physical-placement fields

### If `type` is `PhysicalComponent`

Read it as:

- physical leaf component

Expect:

- `attachedTo`
- `localReferenceFrames`
- `localTimeFrames`
- `position`
- `method`
- no `components`

### If `type` is `PhysicalSystem`

Read it as:

- physical composite system

Expect:

- `attachedTo`
- `localReferenceFrames`
- `localTimeFrames`
- `position`
- `components`
- `connections`

## Process Shortcut Matrix

| If you see | Classify as |
| --- | --- |
| `type: SimpleProcess` | logical leaf |
| `type: AggregateProcess` | logical composite |
| physical-placement fields plus `method` | `PhysicalComponent` |
| physical-placement fields plus `components` | `PhysicalSystem` |

## Aggregate Branch Checklist

If a process has `components`, switch mental mode immediately:

- you are reading a graph, not a leaf node
- each component requires a `name`
- components can be nested processes or a `Link`
- connections use path strings rooted at `components`, `inputs`, `outputs`, `parameters`, or `modes`

Quick test:

- if the object defines internal wiring, it is an aggregate branch

## Physical Branch Checklist

If a process has any of these, it is in the physical half of the family:

- `attachedTo`
- `position`
- `localReferenceFrames`
- `localTimeFrames`

Then split again:

- `method` means physical leaf
- `components` means physical composite

## IO Decision Tree

Start with `type` again.

### Scalar Leaves

If `type` is one of these, classify as scalar:

- `Boolean`
- `Count`
- `Quantity`
- `Time`
- `Category`
- `Text`

Quick differentiators:

- `Quantity` usually carries `uom`
- `Category` usually carries `codeSpace`
- `Time` may carry temporal framing fields
- `Boolean`, `Count`, and `Text` are simpler leaf shapes

### Range Leaves

If `type` is one of these, classify as range:

- `CountRange`
- `QuantityRange`
- `TimeRange`
- `CategoryRange`

Quick differentiator:

- range families use interval-style values rather than a single scalar value

### Record And Tuple Aggregates

If you see `fields`, classify as `DataRecord`.

If you see `coordinates`, classify as `Vector`.

Quick distinction:

- `DataRecord` is a named heterogeneous object
- `Vector` is an ordered coordinate tuple tied to a frame

### Repeated-Element Aggregates

If you see `elementType`, you are in the array or matrix family.

Then split again:

- if there is also frame context like `referenceFrame` or `localFrame`, treat it as `Matrix`
- otherwise treat it as `DataArray`

Practical note:

- both inherit array-style structure from the abstract array base, so `elementCount`, `encoding`, and `values` can appear in the same area

### Tagged Union Aggregate

If you see both `choiceValue` and `items`, classify as `DataChoice`.

Quick rule:

- `DataChoice` is a tagged union, not a record and not a plain array

### Embedded Geometry

If you see `srs` and a GeoJSON-like `value`, classify as `Geometry`.

Also expect:

- geometry constraint fields such as `constraint.geomTypes`

## IO Shortcut Matrix

| If you see | Classify as |
| --- | --- |
| scalar `type` like `Quantity` or `Text` | scalar leaf |
| range `type` like `QuantityRange` | range leaf |
| `fields` | `DataRecord` |
| `coordinates` | `Vector` |
| `elementType` without frame emphasis | `DataArray` |
| `elementType` with frame emphasis | `Matrix` |
| `choiceValue` and `items` | `DataChoice` |
| `srs` and GeoJSON `value` | `Geometry` |

## Five-Second Parse Routine

For a process object:

1. read `type`
2. check for physical-placement fields
3. check for `components`
4. check for `method`

For an IO component:

1. read `type`
2. check for `fields`, `coordinates`, `elementType`, `choiceValue`, or `srs`
3. only after classification, read units, constraints, and nested members

## Connected Systems Shortcut

When reading Connected Systems payloads, this is the shortest useful mapping:

- process branch tells you whether the Part 1 description is a leaf device, logical recipe, or composite system
- IO branch tells you whether a Part 1 input or output is a scalar channel, structured record, repeated array, tagged union, or geometry payload

That usually gives enough context to understand how the Part 1 description should line up with Part 2 datastream and control-stream schemas.