# soksak-contract-registry

Validation and signing rules for exact plugin, sidecar, kit, contract, and spec releases published
by soksak-plugin-registry. Component manifests remain at the versions they declare; immutable
release identities use strict SemVer and each asset URL must use that release's exact tag.

The registry is the current install catalogue, not release history. Each component kind contains
at most one release per component id. Git history and immutable owner releases retain older bytes.
