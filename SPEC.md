# Soksak plugin registry contract 0.0.1

The registry publishes separate plugin, sidecar, kit, contract, and spec release arrays. Every
release has an exact identity, immutable source commit, artifact byte size and SHA-256, its own
manifest, and conformance report references. Release documents do not contain dependency scopes,
install profiles, or provider bindings.

Each array contains at most one current release per component id. Publishing a newer release
replaces that id in the catalogue; immutable owner releases and Git retain history.

Runtime requirements belong to owner manifests. User activation, development paths, and provider
selection belong to settings. Installed paths and artifact digests belong to the Core installation
record. The normative version rules are in `soksak-spec/packages/plugin-spec/docs/VERSIONING.md`.

Each owner repository creates and verifies its release and conformance reports. The registry reads
immutable release documents and never reads or builds owner source trees.

The full registry, validity interval, algorithm and key id are canonicalized and signed
with Ed25519. Consumers pin registry id, key id and public key, reject documents outside the validity
interval, reject sequence rollback and reject different canonical bytes at one sequence.
