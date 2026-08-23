# Soksak plugin registry contract 0.0.1

The registry publishes a plugins array. Every item is one flat immutable release reference and may
project exact direct plugin and sidecar runtime dependencies from that plugin release. It contains
no independent component arrays, install summaries, provider bindings, or release history.

The plugins array contains one current release per plugin id. Publishing a newer release replaces
that id in the catalogue; immutable owner releases and Git retain history.

Runtime dependencies belong to owner manifests and use exact release references. User activation
and development paths belong to the local environment. The normative distribution rules are in
`soksak-spec/packages/plugin-spec/docs/PLUGIN-DISTRIBUTION.md`.

Each owner repository creates and verifies its release and conformance reports. The registry reads
immutable release documents and never reads or builds owner source trees.

The full registry, validity interval, algorithm and key id are canonicalized and signed
with Ed25519. Consumers pin registry id, key id and public key, reject documents outside the validity
interval, reject sequence rollback and reject different canonical bytes at one sequence.
