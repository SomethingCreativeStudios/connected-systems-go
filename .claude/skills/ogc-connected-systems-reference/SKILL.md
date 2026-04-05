---
name: ogc-connected-systems-reference
description: 'Use when working on OGC API - Connected Systems in this repo. Covers Part 1 and Part 2 resource types, schema families, endpoint navigation, relationships, sampling features, procedures, deployments, sub-resources, reparenting, association links, ogc-rel link ownership, field ownership rules, Part 1 to Part 2 field mappings, curated JSON schema lookups, endpoint matrices, SensorML process and IO breakdowns, and UI-oriented rendering guidance for channel schemas and runtime payloads.'
argument-hint: 'Topic to focus on, such as navigation, relationships, links, ownership, endpoint matrix, field mappings, schema lookup, schema reuse, authoring rules, SensorML types, UI cheat sheet, or UI anti-patterns'
user-invocable: true
---

# OGC Connected Systems Reference

Use this skill when you need repository-specific guidance for OGC API - Connected Systems concepts and structure in this workspace.

## When To Use
- Interpreting Part 1 versus Part 2 resources
- Mapping resources to GeoJSON, SensorML, or Part 2 JSON schemas
- Understanding canonical endpoints versus nested traversal paths
- Reasoning about procedures, systems, deployments, sampling features, datastreams, and control streams
- Checking sub-resource creation rules and reparenting constraints
- Distinguishing server-provided `ogc-rel:*` navigation links from client-provided semantic reference links

## Procedure
1. Start with [overview](./references/overview.md) for the resource model and schema families.
2. Read [navigation](./references/navigation.md) when the task involves endpoints, canonical URLs, or traversal flow.
3. Read [relationships](./references/relationships.md) when the task involves FOI, observed properties, controlled properties, or system/procedure/deployment semantics.
4. Read [sub-resources and links](./references/subresources-and-links.md) when the task involves subsystems, subdeployments, sampling feature creation, reparenting, or Part 1 association links.
5. Read [ownership rules](./references/ownership-rules.md) when the task involves server-owned versus client-owned fields, parent-scoped creation, or update/delete semantics.
6. Read [field mappings](./references/field-mappings.md) when the task involves mapping procedure or system inputs and outputs to datastreams, control streams, observations, or commands.
7. Read [schema lookup](./references/schema-lookup.md) when the task needs quick access to copied JSON schema hotspots without reopening the full bundled schema files.
8. Read [endpoint matrix](./references/endpoint-matrix.md) when the task needs a compact create/get/update/delete path map across Part 1 and Part 2.
9. Read [SensorML type guide](./references/sensorml-type-guide.md) when the task involves parsing `AbstractProcess` variants or the `IOComponent` family used in procedures and systems.
10. Read [SensorML nested structures](./references/sensorml-nested-structures.md) when the task involves `configuration`, `modes`, `capabilities`, or `characteristics`.
11. Read [SensorML decision tree](./references/sensorml-decision-tree.md) when the task needs a fast classifier for process and IO variants.
12. Read [Part 1 to Part 2 schema reuse](./references/part1-part2-schema-reuse.md) when the task involves reusing SensorML IOComponent definitions as datastream or control-stream schemas, especially for UI automation.
13. Read [authoring rules](./references/authoring-rules.md) when the task involves create forms, editable fields, or deciding which payload fields should be omitted because they are server-derived.
14. Read [UI cheat sheet](./references/ui-cheat-sheet.md) when the task involves mapping IOComponent variants to concrete form controls, tables, charts, map widgets, or read-only inspectors.
15. Read [UI anti-patterns](./references/ui-anti-patterns.md) when the task involves reviewing or designing forms and viewers to avoid model-breaking UX choices.

## Repository Scope
- Part 1 OpenAPI source: `schemas/openapi/connected-systems-Part 1 All.yaml`
- Part 2 OpenAPI source: `schemas/openapi/*.yaml`
- Part 1 schemas: `schemas/geojson/` and `schemas/sensorml/`
- Part 2 schemas: `schemas/json/`

## Notes
- This skill is repository-specific and reflects both the normative Part 1 standard and the project's working interpretation where that interpretation is stricter or more implementation-oriented.
- When the task depends on exact normative behavior, prefer the Part 1 standard and local OpenAPI files over the summary text.
