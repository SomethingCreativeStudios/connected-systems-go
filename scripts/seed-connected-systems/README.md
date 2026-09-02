# Connected Systems Seeder

This command creates coherent OGC API - Connected Systems Part 1 and Part 2
resources through the configured HTTP endpoint. It never connects to the
database and it never deletes existing resources.

```sh
go run . -config ./config.example.yaml
```

`mode: seed` creates one namespaced graph. `namespace` remains the stable
ownership tag and each run receives a fresh UTC `run_id` by default, so a
second run is additive even with the same `random_seed`. Set `run_id` only when
you need fixed fixture identifiers; rerunning that explicit ID will collide
with the first run rather than overwrite it. The seeder captures IDs from API `Location`
headers and uses them for every system/procedure, sampling-feature/generic-
feature, dynamic-data, and deployment relation. `mode: observe` repeatedly
selects a random batch of compatible existing datastreams and posts
schema-correct JSON observations until interrupted.

The API currently ingests Part 2 resources with JSON envelopes. The seeder
therefore models JSON, SWE-JSON, and protobuf schemas with JSON-shaped result
values; it does not claim to send raw SWE CSV or protobuf bytes.
