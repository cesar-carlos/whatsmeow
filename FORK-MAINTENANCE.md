# Evolution Go interactive compatibility patch

Base: `b572e5bcb92bbc285b68cb6d6540da3093330e04` from tulir/whatsmeow.
This fork keeps that base and changes only interactive node construction.
References: tulir/whatsmeow #1235 (felps-dev), #1221 (gsdev-br), #1146 (TobyG74).
The original MPL-2.0 license and copyright notices remain applicable.

Changes: unwrap bounded/nil-safe payloads, native-flow/list nodes, explicit-node
precedence without duplicates, and no business nodes on interactive responses.
Group encryption, session schema and unrelated upstream changes are untouched.
Observed node attributes are experimental protocol compatibility, not an official
specification. Unit tests do not prove rendering on WhatsApp devices.

Run `go build ./...`, `go vet ./...`, `go test -timeout 5m ./...` and
`go test -race -timeout 5m ./...`. Keep application references pinned to a full
immutable commit/pseudo-version. On future upstream updates, retain regression
tests and retire this patch only after equivalent behavior and device validation.
