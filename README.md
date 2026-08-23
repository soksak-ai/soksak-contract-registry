# soksak-contract-registry

Authentication and continuity rules for the Soksak plugin registry. The registry contains one
current release reference per plugin and projects only that plugin's exact direct runtime
dependencies. Component details remain in immutable owner releases.

The registry is authenticated by definition. Consumers pin its identity, key ID, and public key,
then reject invalid signatures, expiry, rollback, and equivocation.
