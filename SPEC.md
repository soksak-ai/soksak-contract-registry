# Soksak plugin registry contract 0.0.1

The registry exposes plugins to users and retains sidecar and kit releases only as dependency
closure nodes. Every release has exact identity, immutable source commit, exact dependencies,
targeted archives containing soksak-unit.json and conformance report references.

A profile selects one plugin root and explicit provider bindings inside that plugin's exact closure.
The plugin manifest remains provider-agnostic. No provider is selected by install order, directory
order, name convention or fallback.

Each owner repository creates and verifies its release and conformance reports. The registry reads
immutable release documents and never reads or builds owner source trees.
