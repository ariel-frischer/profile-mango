# Architecture

Profile Mango separates portable policy from target-specific configuration effects.
The [adapter architecture](dev/adapter-architecture.md) describes rendering, and the
[project scope](dev/project-scope.md) records current shipped capabilities.

## Shared ownership

- `pkg/profilemango` owns pure canonical parsing, resolution, and resource identity.
- `pkg/render` and `pkg/adapters/<target>` own deterministic inert previews and exact-version evidence.
- `pkg/install` owns one target-neutral `Adapter`/`Patch` boundary, deterministic plans, capability gates, and application orchestration. Target-specific install adapters supply lossless bounded patches, not replacement preview files.
- `internal/installfs` owns bounded filesystem inspection, transactions, backups, journals, and guarded recovery.
- `cmd/profile-mango/install.go` owns shared target selection, human/JSON output, interactive consent, and hash-bound noninteractive apply. There is no separate installer command or transaction engine per target.

## Safety and evidence

Planning is read-only. Apply requires explicit consent bound to the displayed plan.
Backups default on, unrelated state remains user-owned, and stale state or unknown
required properties block mutation. Credentials and target processes are outside
ordinary install execution. Native qualification is a separate disposable,
credential-free, network-blocked developer workflow. Native parsing, effective
configuration, delivery, and runtime enforcement are distinct evidence levels.
Only exact subsets recorded in the target evidence ledger are install-capable.
Future target promotion requires evidence, not merely a new registry entry.

## Parallel installation work

Each target Bead owns its target installer, preservation tests, exact-version
qualification, and target evidence. The shared CLI Bead owns automation and
confirmation semantics. Root owns registry integration, shared documentation,
changelog, cross-target acceptance, and serial landing. Target lanes consume the
existing `Adapter`/`Patch` contract and must escalate shared-contract changes rather
than independently changing it. Experimental Jcode remains outside public support.
