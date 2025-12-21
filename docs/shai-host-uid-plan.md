# Align container user UID with host caller

Goal: run the in-container user (default name `shai`) with the same UID (and ideally GID) as the host user invoking `shai`, so bind-mounted files retain sensible ownership.

## Current behavior (code)
- `internal/shai/runtime/ephemeral_runner.go`: the container is started as root; the bootstrap receives `--user` (default `shai`) but no UID/GID hint. `UserOverride` only changes the username.
- `internal/shai/runtime/bootstrap/bootstrap.sh`: creates the target user if missing via `useradd`, optionally honoring `REQUESTED_DEV_UID=${DEV_UID:-4747}`. If the user already exists, it keeps the existing UID. Egress iptables rules apply to `DEV_UID` (set after `id -u $TARGET_USER`), so they mirror whatever UID the target user ends up with.
- Result: the target user typically has UID 4747 (or the image default), which often differs from the host caller’s UID.

## Implementation steps
1) Capture host caller IDs in Go
   - In the runtime layer (e.g., new helper in `internal/shai/runtime/env.go` or near runner init), read the host UID/GID (`os.Getuid`, `os.Getgid`, fallback to `os/user.Current().Uid`). Keep the values as strings to avoid cross-platform issues.
   - Plumb these into `EphemeralConfig`/`EphemeralRunner` (new fields) and inject them into the container env in `buildDockerConfigs` (e.g., `DEV_UID`, `DEV_GID`). Keep existing `UserOverride` semantics for the username.

2) Teach bootstrap to honor requested UID/GID robustly
   - Update `internal/shai/runtime/bootstrap/bootstrap.sh` to accept `DEV_UID`/`DEV_GID` (renaming `REQUESTED_DEV_UID` accordingly) and to reconcile the target user when it already exists:
     - If the user is missing: create it with `useradd -u <DEV_UID>` and, when provided, align the primary group (create/use group with `DEV_GID`).
     - If the user exists but has a different UID/GID and we are root: adjust via `usermod`/`groupmod` (handling collisions where the UID/GID is already taken by another entry—either reuse that account if it matches the requested UID/GID, or fallback with a warning).
     - If not running as root and the UID mismatches: emit a clear error or warning plus fallback path (decision below).
   - Ensure `DEV_UID`/`DEV_GID` exported after reconciliation so iptables rules in `dev_egress_setup` operate on the matched IDs.

3) Guard tricky cases and compatibility
   - Decide on behavior when the host UID is 0: keep the current sandbox UID (4747) to avoid applying egress rules to root, or allow root with a clear warning and loosen egress rules for UID 0.
   - Handle images that already have the requested UID on a different username: prefer reusing that user vs. mutating the target username, and document the choice.
   - Preserve an escape hatch to the old behavior (e.g., env flag to skip host UID sync) in case consumers rely on fixed UID 4747.

4) Update surfaced behavior and docs
   - Reflect the new default in `docs/shai-example-config.yaml` and any README snippets mentioning user defaults.
   - Consider logging the resolved UID/GID in verbose mode (e.g., tweak the start marker in `buildStartMarker` or bootstrap debug logs) for visibility.

5) Tests
   - Add unit coverage in Go (e.g., new test in `internal/shai/runtime/ephemeral_runner_test.go`) to assert `DEV_UID`/`DEV_GID` env injection.
   - Add shell-level coverage for bootstrap user reconciliation (can be a small bash test harness similar to other bootstrap tests) to prove UID alignment and collision handling.
   - Exercise a happy-path integration (existing `ephemeral_e2e_test.go`) verifying `id -u` inside the container matches the host UID.

## Open decisions/risks to settle
- Policy for host UID 0 and for images without `useradd`/`usermod` (fallback vs. hard error).
- Exact handling when the desired UID/GID already belongs to another account: reuse, remap, or fail loudly.
- Whether to sync only UID or both UID and primary GID by default.
