# Soksak plugin registry contract 0.0.1

The registry exposes plugins to users and retains sidecar and kit releases only as dependency
closure nodes. Plugin, sidecar and kit releases are separate document types. Every release has exact
identity, immutable source commit, exact dependencies, targeted archives containing its own
kind-specific manifest and conformance report references.

Every dependency declares runtime or build scope. Runtime dependencies enter the installer closure
and settings composition. Build dependencies record reproducibility of the owner artifact and are
not installed separately. A kit may use either scope; its kind does not imply its scope.

A profile selects one plugin root and explicit provider bindings inside that plugin's exact closure.
The plugin manifest remains provider-agnostic. No provider is selected by install order, directory
order, name convention or fallback.

The runtime install closure is the plugin's runtime dependency closure plus every explicitly bound
provider and each provider's runtime dependencies. A binding consumer must already be inside the
plugin root closure; a binding provider need not be a direct plugin dependency.

Each owner repository creates and verifies its release and conformance reports. The registry reads
immutable release documents and never reads or builds owner source trees.

The full registry, profiles, validity interval, algorithm and key id are canonicalized and signed
with Ed25519. Consumers pin registry id, key id and public key, reject documents outside the validity
interval, reject sequence rollback and reject different canonical bytes at one sequence.
