# Koolo DiobyteVersion — Fork Differences Only

This repository is a **fork** of [kwader2k/koolo](https://github.com/kwader2k/koolo). The contents of this README are
intentionally limited to **changes introduced in this fork**; everything not mentioned here is inherited from the
upstream project.

## What's different from upstream

### Scheduler activation & dormant UI (`internal/bot/scheduler.go`, `internal/server/`)

- Per-supervisor scheduler activation tracking with mutex-guarded `ActivateCharacter`/`DeactivateCharacter`/`IsActivated` helpers.
- Scheduler is automatically activated on non-manual starts (including auto-start flow) and deactivated on Stop.
- Manual-mode supervisors are skipped by the scheduler to avoid interference.
- HTTP API exposes `Activated`, `ScheduleSummary`, and `SimpleStopTime` fields in scheduler status responses;
  a `scheduleSummary()` helper produces human-readable summaries for simple/duration/time-slots modes.
- Dashboard CSS adds dormant and header-badge states; dashboard JS shows a compact scheduler badge, a dormant
  summary when the scheduler is enabled but not yet activated, improved countdown rendering with
  `countdown-live` elements, and a 30-second auto-refresh to keep countdowns accurate.

### Human-like timing & cursor randomness (`internal/utils/sleep.go`, multiple action files)

- Click positions for buffs, CTA casts, and item pickup are jittered with small random offsets.
- Pickup spiral coordinates receive per-step random offsets.
- Attack and timing sleeps have +/- jitter added to break metronomic cast intervals.
- Walk-polling replaces flat uniform sampling with **Gamma-based** sampling (`RandGammaDurationMs`).
- Idle gaps use **log-normal** sampling (`RandLogNormal`).
- Idle cursor-wander move counts use a skewed geometric-like distribution.
- `utils/sleep.go` gains `RandGammaDurationMs` and `RandLogNormal`; `sessionMu` upgraded to `RWMutex`
  with `RLock` in `sessionFatigue` for safer concurrent reads.

### Bot state, stash safety & logging refactors (`internal/action/`, `internal/bot/`, `internal/context/`, `internal/game/`)

- **Per-supervisor monster-state tracking** in `action/step/attack.go`: monster state maps are keyed
  by bot name (mutex-guarded) to eliminate `UnitID` collisions between concurrent supervisors.
- **Stash gold slice guards** in `action/stash.go`: length checks before indexing `StashedGold`,
  safe total computation, removal of noisy debug prints.
- `maxInteractions` in `action/step/pickup_item.go` made function-local so high-attempt modes get
  extra tries and the global variable is removed.
- `bot/bot.go` `shouldReturnToTown` simplified with early returns; never returns if already in town
  or in UberTristram.
- `bot/manager.go` preserves the `Runtime` field correctly across hot-reloads.
- `bot/single_supervisor.go` removes the erroneous reset of `FailedToCreateGameAttempts` in the
  modal-absent branch.
- `context/context.go` `Get()` panics on unregistered goroutines to surface misuse; `getGoroutineID`
  uses a smaller buffer and faster numeric parse.
- `game/manager.go` and `game/packet_sender.go` replace `fmt` debug prints with structured `slog` logging.
- `character/sorceress_leveling.go` debug messages converted to `ctx.Logger.Debug` with context fields.

### Safety guards for nil/bounds panics (`internal/run/`, `internal/action/`, `internal/town/`)

- Bounds checks added before accessing `NPC.Positions[0]` across `interaction.go`, `anya.go`,
  `quests.go`, `cave.go`, `bone_ash.go`, `jail.go`, `izual.go`, `countess.go`, `A1.go`.
- `action/move.go` shrine lookup: fix shadowed variable that caused the best-shrine result to always
  be `nil`; result is now stored in a scoped variable and returned after the loop.
- `action/repair.go`: remove unused import alias; call `context.Get()` directly.
- `action/vendor.go`: replace removed `botCtx` alias with `context.Get()`; adjust Jamella key
  sequence to skip `VK_DOWN`; enforce `MaxGameLength` only when it is greater than zero.

### Portal/waypoint refresh guards (`internal/run/leveling_act4.go`, `internal/run/leveling_act5.go`)

- After sending the Harrogath portal key sequence, wait, refresh game data, and re-query the portal;
  return an error if it is still missing (both act-4 portal locations).
- Act-5 waypoint usage guarded by existence check before calling `MoveToCoords` to prevent a nil
  dereference.

---

> For a file-level diff against upstream run:
> ```sh
> git fetch upstream && git diff --stat upstream/main
> ```

## Notes

- The game must still be set to **English**. 1280x720 windowed mode and LOD 1.13c are required as usual.
- All other documentation (installation, usage, pickit rules, etc.) is unchanged from upstream — refer to
  the [kwader2k/koolo](https://github.com/kwader2k/koolo) README for full details.

---

*This README covers only the modifications made in this fork. See the upstream project for the full Koolo documentation.*