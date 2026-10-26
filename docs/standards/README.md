# Part 3 snapshot

Source: https://docs.ogc.org/DRAFTS/23-003r0.html

Retrieved 2026-10-14. The served draft was last modified 2026-10-08.
SHA-256 of `23-003r0.html`:
`1baa5807ead76c7205c851816211a0ddde4baccd44cb2096a499e4055abe6d57`.
The original OGC copyright and license notices remain in the HTML.

This snapshot pins the requirements used by `docs/part3-mqtt.md`; a future revision
of the mutable draft URL does not silently change this implementation's target.

The embedded broker is `github.com/mochi-mqtt/server/v2` v2.7.9, with
`github.com/cockroachdb/pebble` v1.1.5 for broker state. Exact module checksums are
in `go.sum`; Go verifies them while downloading the dependencies. The broker
release is unmodified. The CS API Go storage adapter uses Mochi's storage record
types and closes iterators that the bundled v2.7.9 Pebble hook leaks.

The C plugin, custom Mosquitto image, private ingress listener and HMAC transport
metadata were removed. Existing deployment-owned Mosquitto services for other
applications remain independent; they are not part of CS API conformance.
