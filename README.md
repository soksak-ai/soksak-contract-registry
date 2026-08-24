# soksak-contract-registry

Authentication and continuity rules for the Soksak plugin registry. The registry contains one
current release reference per plugin and projects only that plugin's exact direct runtime
dependencies. Component details remain in immutable owner releases.

The registry is authenticated by definition. Consumers pin its identity, key ID, and public key,
then reject invalid signatures, expiry, rollback, and equivocation.

## Verification

```sh
make verify
```

`go.mod` is the exact Go owner. This gate verifies only catalogue authentication and continuity
rules; the Registry product verifies its own publication and installed catalogue separately.
