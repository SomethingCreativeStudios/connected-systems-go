# SensorML Type Guide

## Purpose

This guide is for parsing the large SensorML process and IO type unions used by `Procedure` and `System` schemas in this repo.

The goal is to answer two questions quickly:

- which fields are shared by all process variants?
- which fields identify the specific process or IO variant?

## AbstractProcess Family

In the bundled SensorML schemas, `Procedure` and `System` both resolve to a process-family union with four main concrete variants:

- `SimpleProcess`
- `AggregateProcess`
- `PhysicalComponent`
- `PhysicalSystem`

Think of them as a 2 x 2 split:

- simple versus aggregate
- logical versus physical

## Common Fields Shared Across Process Variants

The shared process base is large, but the most important common fields are:

- `type`
- `id`
- `description`
- `uniqueId`
- `label`
- `lang`
- `keywords`
- `identifiers`
- `classifiers`
- `validTime`
- `securityConstraints`
- `legalConstraints`
- `characteristics`
- `capabilities`
- `contacts`
- `documents`
- `history`
- `inputs`
- `outputs`
- `parameters`
- `modes`
- `configuration`

Practical parser rule:

- first read these as the common descriptive shell
- then switch on `type` for the variant-specific block

## Variant 1: SimpleProcess

Type discriminator:

- `type: SimpleProcess`

Unique fields:

- `method`

Meaning:

- one process definition without embedded components
- good for a single algorithm, workflow, or one-node device behavior description

Parser shortcut:

- if there is `method` and no `components`, this is the simplest process branch

## Variant 2: AggregateProcess

Type discriminator:

- `type: AggregateProcess`

Unique fields:

- `components`
- `connections`

Meaning:

- a composite process graph
- components are named child processes or links to processes
- connections explicitly wire outputs, inputs, parameters, or modes across those components

Important details:

- component entries require a `name`
- each component can itself be a `SimpleProcess`, `AggregateProcess`, `PhysicalComponent`, `PhysicalSystem`, or a `Link`
- connection paths use strings like `components/...`, `inputs/...`, `outputs/...`, `parameters/...`, or `modes/...`

Parser shortcut:

- if you see `components`, treat the object as a graph container rather than a leaf process

## Variant 3: PhysicalComponent

Type discriminator:

- `type: PhysicalComponent`

Shared-with-physical fields:

- `attachedTo`
- `localReferenceFrames`
- `localTimeFrames`
- `position`

Unique leaf-style field:

- `method`

Meaning:

- a physical leaf device or component with physical placement context
- still leaf-like, because it does not introduce `components`

Parser shortcut:

- if it has physical placement fields plus `method`, it is the physical analogue of `SimpleProcess`

## Variant 4: PhysicalSystem

Type discriminator:

- `type: PhysicalSystem`

Shared-with-physical fields:

- `attachedTo`
- `localReferenceFrames`
- `localTimeFrames`
- `position`

Aggregate-style fields:

- `components`
- `connections`

Meaning:

- a composite physical system made of named sub-components
- the physical analogue of `AggregateProcess`

Parser shortcut:

- if it has physical placement fields plus `components`, it is the physical composite branch

## Fast Process Matrix

| Type | Physical Context | Components | Method | Best Mental Model |
| --- | --- | --- | --- | --- |
| `SimpleProcess` | no | no | yes | logical leaf |
| `AggregateProcess` | no | yes | no | logical composite |
| `PhysicalComponent` | yes | no | yes | physical leaf |
| `PhysicalSystem` | yes | yes | no | physical composite |

## Process Parsing Order

Use this order when reading a large SensorML process object:

1. read `type`
2. read common descriptive fields
3. inspect `inputs`, `outputs`, `parameters`
4. if `components` exists, switch to composite parsing
5. if physical fields exist, read `attachedTo`, frames, and `position`
6. if `method` exists, treat it as a leaf-style behavior description

## IOComponent Family

The IO layer is also a union. In the bundled schemas this appears as `IOComponentChoice`.

At the highest useful level, the choices break into:

- scalar data components
- scalar range components
- aggregate components
- geometry-like components

## Common Fields Shared Across Many IO Variants

The most important common IO fields are:

- `type`
- `definition`
- `label`
- `description`

Frequent optional shared fields:

- `updatable`
- `optional`
- `value`
- `constraint`
- `nilValues`

Specialized but frequent fields:

- `uom`
- `referenceFrame`
- `localFrame`
- `codeSpace`

Parser rule:

- `definition` and `label` are the main semantic anchors
- `type` chooses the structural branch

## Scalar IO Variants

The scalar branch includes these common leaf types:

- `Boolean`
- `Count`
- `Quantity`
- `Time`
- `Category`
- `Text`

Main differences:

- `Boolean`: boolean value, no unit
- `Count`: integer value, count-style constraint set
- `Quantity`: numeric value with `uom`
- `Time`: time value with `uom`, optional `referenceTime` or `localFrame`
- `Category`: token-like categorical value with `codeSpace`
- `Text`: free text, optional token constraint set

## Scalar Range Variants

The range branch includes:

- `CountRange`
- `QuantityRange`
- `TimeRange`
- `CategoryRange`

Main pattern:

- same semantic family as their scalar counterparts
- but `value` becomes a two-element array representing a range

Main differences:

- `QuantityRange`: requires `uom`
- `TimeRange`: supports `referenceTime`, `localFrame`, and temporal `uom`
- `CategoryRange`: uses `codeSpace`
- `CountRange`: integer-pair version of count

## Aggregate IO Variants

The aggregate branch includes:

- `DataRecord`
- `Vector`
- `DataArray`
- `Matrix`
- `DataChoice`

### DataRecord

Unique fields:

- `fields`

Meaning:

- named heterogeneous group of sub-components
- best for structured payloads such as command records or observation result records

### Vector

Unique fields:

- `referenceFrame`
- `localFrame`
- `coordinates`

Meaning:

- ordered coordinate set tied to a frame of reference
- best for positions, directions, velocities, and other frame-aware numeric tuples

### DataArray

Unique fields:

- `elementType`
- inherited array-count structure from the abstract array base

Meaning:

- repeated homogeneous component values
- best for trajectories, time series blocks, raster-like samples, and repeated records

### Matrix

Unique fields:

- `referenceFrame`
- `localFrame`
- abstract-array fields like `elementType`

Meaning:

- framed array structure used for transforms or matrix-valued quantities

### DataChoice

Unique fields:

- `choiceValue`
- `items`

Meaning:

- disjoint union of alternative component types
- `choiceValue` tells you which branch is active for a value instance

Parser rule:

- if you see `items` plus `choiceValue`, do not parse it like a record; it is a tagged union

## Geometry-Like IO Variant

The IO family also includes `Geometry`.

Unique fields:

- `constraint.geomTypes`
- `srs`
- `value` as embedded GeoJSON geometry

Meaning:

- geometry embedded as a data component rather than as the outer feature geometry

## Fast IO Matrix

| Type Family | Common Meaning | Distinguishing Fields |
| --- | --- | --- |
| Scalar | one value | `value`, maybe `uom` or `codeSpace` |
| Range | two-value interval | `value` as 2-element array |
| `DataRecord` | named heterogeneous structure | `fields` |
| `Vector` | framed coordinate tuple | `coordinates`, `referenceFrame` |
| `DataArray` | homogeneous repeated values | `elementType` |
| `Matrix` | framed repeated values | `elementType`, `referenceFrame` |
| `DataChoice` | tagged union | `choiceValue`, `items` |
| `Geometry` | embedded spatial geometry | `srs`, GeoJSON `value` |

## Practical Parsing Strategy

When an IO component is deeply nested, use this order:

1. read `type`
2. read `definition` and `label`
3. decide leaf versus aggregate
4. if leaf, inspect `value`, `uom`, `codeSpace`, `constraint`, `nilValues`
5. if aggregate, inspect `fields`, `coordinates`, `elementType`, or `items`
6. if `referenceFrame` or `localFrame` exists, treat the value as frame-aware rather than a plain tuple

## Why This Matters For Connected Systems

These types are the bridge between Part 1 SensorML descriptions and Part 2 runtime stream schemas.

In practice:

- `inputs`, `outputs`, and `parameters` on procedures and systems are built from these IO component families
- `Datastream.resultSchema` and `ControlStream.parametersSchema` often mirror the same structural ideas
- understanding the union once makes both SensorML metadata and Part 2 schemas easier to read