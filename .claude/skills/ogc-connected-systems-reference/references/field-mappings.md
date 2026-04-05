# Field Mappings

## Purpose

This reference ties the Part 1 descriptive model to the Part 2 runtime model.

The most important question it answers is:

- how do procedure and system inputs or outputs line up with datastreams, observations, control streams, and commands?

## Mental Model

Use this graph:

- `Procedure` defines the reusable channels and data meaning
- `System` instantiates those channels in a concrete asset
- `Datastream` binds to one system output
- `Observation` carries one runtime result inside that output channel
- `ControlStream` binds to one system input
- `Command` carries one runtime actuation request inside that input channel

## Output Mapping

The practical output-side chain is:

- procedure output definition
- concrete system output definition
- `Datastream.outputName`
- datastream schema
- datastream `observedProperties`
- observation `result`

In this project, `Datastream.outputName` is the key join field.

It should map a datastream to:

- the output name declared by the concrete system instance
- and the corresponding reusable output declared by the system's procedure

## Input Mapping

The practical input-side chain is:

- procedure input definition
- concrete system input definition
- `ControlStream.inputName`
- control-stream schema
- control-stream `controlledProperties`
- command `parameters`

In this project, `ControlStream.inputName` is the key join field.

It should map a control stream to:

- the input name declared by the concrete system instance
- and the corresponding reusable input declared by the system's procedure

## Part 1 To Part 2 Output Mapping

Map these concepts together:

- procedure output definition -> reusable semantic description of an observable channel
- system output definition -> concrete instance channel available on one system
- `Datastream.outputName` -> selected runtime binding to that channel
- datastream `schema.resultSchema` -> shape of each observation payload on that channel
- datastream `observedProperties` -> property identifiers exposed by that channel
- observation `result` -> one concrete runtime value instance emitted by that channel

Practical rule:

- if the datastream says it is bound to output `temp`, then the observed properties and result schema should match the `temp` output defined on the system and ultimately on the procedure

## Part 1 To Part 2 Input Mapping

Map these concepts together:

- procedure input definition -> reusable semantic description of a controllable channel
- system input definition -> concrete instance channel available on one system
- `ControlStream.inputName` -> selected runtime binding to that channel
- control-stream `schema.parametersSchema` -> shape of each command payload on that channel
- control-stream `controlledProperties` -> property identifiers affected by that channel
- command `parameters` -> one concrete runtime actuation payload submitted to that channel

Practical rule:

- if the control stream says it is bound to input `ptz`, then the controlled properties and command parameter schema should match the `ptz` input defined on the system and ultimately on the procedure

## Schema Mapping

## Shared Schema Family Rule

The same SensorML `IOComponent` family used by Part 1 `inputs`, `outputs`, and `parameters` is also the structural family reused by Part 2 stream schema definitions.

Practical interpretation:

- a system or procedure output can often be reused almost directly as a datastream `schema.resultSchema`
- a system or procedure input can often be reused almost directly as a control-stream `schema.parametersSchema`
- a system or procedure parameter can often be reused as a datastream or command parameter schema when runtime parameter blocks are needed

This is the most useful bridge for UI automation because one IOComponent parser can drive both:

- Part 1 design-time forms and inspectors
- Part 2 runtime stream forms and payload viewers

### Datastream Schema

The datastream-level schema resource describes the contract for nested observations.

Important fields in the local JSON schema are:

- `obsFormat`
- `parametersSchema`
- `resultSchema`
- `resultLink`

Mapping:

- `resultSchema` describes the structure of `Observation.result`
- `parametersSchema` describes the structure of `Observation.parameters`
- `resultLink` is used when the observation result is delivered out-of-band through `result@link`

Important reuse note:

- the underlying schema family here is the same SWE component vocabulary used for Part 1 outputs and parameters, so a datastream can often directly operationalize a system or procedure channel definition

### ControlStream Schema

The control-stream-level schema resource describes the contract for nested commands and related results.

Important fields in the local JSON schema are:

- `commandFormat`
- `parametersSchema`
- `resultSchema`
- `feasibilityResultSchema`

Mapping:

- `parametersSchema` describes the structure of `Command.parameters`
- `resultSchema` describes inline `CommandResult` content for regular commands
- `feasibilityResultSchema` describes inline result content for feasibility responses

Important reuse note:

- the underlying schema family here is the same SWE component vocabulary used for Part 1 inputs and parameters, so a control stream can often directly operationalize a system or procedure channel definition

## Property Mapping

### Observed Properties

Use this chain:

- procedure output meaning
- system output meaning
- datastream `observedProperties`
- observation result fields

Project rule:

- a datastream may summarize its observed properties from nested observations
- but that summary still has to stay consistent with the selected system output and corresponding procedure output

### Controlled Properties

Use this chain:

- procedure input meaning
- system input meaning
- control-stream `controlledProperties`
- command parameter fields

Project rule:

- a control stream may summarize its controlled properties from command handling context
- but that summary still has to stay consistent with the selected system input and corresponding procedure input

## Feature Of Interest Mapping

The cleanest FOI chain in this project is:

- system
- sampling feature attached to that system
- `sampledFeature@link` on the sampling feature
- stream `samplingFeature@link`
- stream `featureOfInterest@link`

Interpretation:

- `samplingFeature@link` is the immediate sample context of a stream, observation, or command
- `featureOfInterest@link` is the ultimate domain feature reached through that sample

Practical derivation rule:

- `featureOfInterest@link` should be derived from the linked sampling feature's `sampledFeature@link`

## Procedure Overrides At Item Level

Both observations and commands can carry `procedure@link` locally.

Use that as a narrow override or specialization only when needed.

Default interpretation:

- datastream or control-stream context gives the main procedure association
- item-level `procedure@link` is only needed when a specific observation or command was handled by a more precise method than the parent stream advertises

## Deployment Mapping

Deployment context is usually one level removed from the item payloads.

Use this chain:

- deployment
- deployed system
- datastream or control stream attached to that system
- observations or commands carried by the stream

Project interpretation:

- stream deployment links are usually derived by the server from the attached system context rather than authored independently on every stream or item

## Validation Checklist

When checking whether a Part 2 resource is semantically correct in this project, ask:

- does `outputName` or `inputName` map to a real system channel?
- does that channel map cleanly to the linked procedure definition?
- do the property lists match the selected channel?
- do observation or command payloads validate against the parent stream schema?
- do stream FOI links resolve through the sampling-feature chain?