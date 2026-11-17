# Evolution Go interactive compatibility patch

Base: `54038813c47f3f351eb6fcaafab0e1b99d676850` from tulir/whatsmeow,
matching Evolution Go 0.7.3 (reviewed 2026-11-17).
This fork preserves interactive node construction and status ACK regressions.
References: tulir/whatsmeow #1235 (felps-dev), #1221 (gsdev-br), #1146 (TobyG74).
The original MPL-2.0 license and copyright notices remain applicable.

Changes: unwrap bounded/nil-safe payloads, native-flow/list nodes, explicit-node
precedence without duplicates, and no business nodes on interactive responses.
The official base's schema v16, protocol and dependency updates are retained.
Observed node attributes are experimental protocol compatibility, not an official
specification. Unit tests do not prove rendering on WhatsApp devices.

Follow-up: list business-node attributes now preserve the protobuf list type
(`single_select` or `product_list`), rather than advertising every menu as a
product catalog. Attribute lookup shares the bounded, nil-safe unwrapping path.
Regression tests exercise all supported wrappers, empty/cyclic/deep envelopes,
selection IDs/context, response formats and duplicate explicit business nodes.
This checks internal consistency; device/server acceptance still needs Android,
iOS and Web validation, including real row selection and received response IDs.

Run `go build ./...`, `go vet ./...`, `go test -timeout 5m ./...` and
`go test -race -timeout 5m ./...`. Keep application references pinned to a full
immutable commit/pseudo-version. On future upstream updates, retain regression
tests and retire this patch only after equivalent behavior and device validation.

## Status ACK (Evolution Go F-025)

Production used `0a13d3117a80f33e1325e967e0a4cae800bc09f2` on the previous
base. Official commit `a2381054887e16d22428430fdc4d2b027a7f29a7` already
provides the same handler, ACK for non-newsletter status and newsletter marker.
The merged client/message implementation equals the official base; no duplicate
handler is added. `status_ack_test.go` preserves production's regression and
also verifies the ACK attempt via the disconnected transport error. It covers
media/text status without chat events and newsletter delivery with the marker.

Validation uses Go 1.26.10 Linux, GOTOOLCHAIN=local, build/vet/test/race.
No WhatsApp connection, real message, production database or deployment is used.
