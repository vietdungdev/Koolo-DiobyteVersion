# Koolo DiobyteVersion — Fork Differences Only

This repository is a **fork** of [kwader2k/koolo](https://github.com/kwader2k/koolo). The contents of this README are
intentionally limited to **changes introduced in this fork**; everything not mentioned here is inherited from the
upstream project.

## Build requirements (changed from upstream)

The upstream project builds with **Go 1.23**. This fork requires newer toolchain versions — it
**will not compile** with the original Go and Garble versions.

| Tool       | Upstream version | This fork   | Install command                                 |
|------------|-----------------|-------------|-------------------------------------------------|
| **Go**     | 1.23            | **1.25.7**  | Download from [go.dev/dl](https://go.dev/dl/)   |
| **Garble** | (any)           | **v0.15.0** | `go install mvdan.cc/garble@v0.15.0`          |

**Before building this fork you must:**

1. **Uninstall Go 1.24 / 1.23** (or whichever version you currently have).
2. **Install Go 1.25.7** — the `go.mod` directive is `go 1.25.7`; older compilers will refuse to
   build or produce subtle incompatibilities.
3. **Install Garble v0.15.0** — earlier Garble releases are incompatible with Go 1.25 and will fail
   during the obfuscation pass.  Run:
   ```
   go install mvdan.cc/garble@v0.15.0
   ```
4. Verify with `go version` (should print `go1.25.7`) and `garble version` (should print
   `v0.15.0`).

> The included `better_build.bat` checks both versions automatically and offers to install them if
> they are missing or outdated.

---

## What's different from upstream

**86 Go files changed — +2,032 / -671 lines** across the categories below.

### 1. SigmaDrift mouse movement (`internal/game/sigmadrift.go`, `mouse.go`, `memory_injector.go`)

Upstream `MovePointer` teleports the cursor instantly — a single `CursorPos` call followed by
`WM_NCHITTEST` / `WM_SETCURSOR` / `WM_MOUSEMOVE`. This fork replaces that with a full
biomechanically-grounded trajectory system ported from
[ck0i/SigmaDrift](https://github.com/ck0i/SigmaDrift) (`sigmadrift.go` — **372 lines, entirely new**).

Key differences from upstream:

- **Sigma-lognormal velocity primitives** (Plamondon's Kinematic Theory): the cursor follows a
  lognormal CDF position profile with a configurable peak-time ratio, producing a natural
  acceleration/deceleration bell curve instead of an instantaneous jump.
- **Two-phase surge architecture**: a ballistic primary stroke covers 92–97 % of the distance,
  followed by 0–2 corrective sub-movements that simulate the undershoot/overshoot adjustment humans
  make when landing on a target.
- **Ornstein-Uhlenbeck lateral drift**: mean-reverting stochastic hand drift is applied
  perpendicular to the movement axis via Euler-Maruyama integration, modelling natural hand sway.
- **Physiological tremor (8–12 Hz)**: sinusoidal tremor is overlaid on each axis, with amplitude
  suppressed at high cursor speed (proprioceptive gating).
- **Signal-dependent noise (Harris-Wolpert)**: Gaussian noise proportional to instantaneous motor
  command magnitude is added each sample, reproducing the known relationship between movement speed
  and endpoint variability.
- **Gamma-distributed inter-sample timing**: sample intervals are drawn from a Gamma distribution
  (shape 3.5, mean ~7.8 ms) rather than a fixed polling rate, eliminating the constant-dt
  fingerprint that characterises bot-generated event streams.
- **Lateral curvature profile**: a `s`^2`(1-s)`^3 arc scaled by movement angle (vertical > horizontal)
  adds the gentle lateral bow observed in human pointing movements.
- **Micro-correction pass (12 % probability)**: after the trajectory lands, there is a random chance
  of a 2–5 px overshoot followed by a brief dwell and re-aim to the exact target, breaking the
  otherwise perfectly-sharp endpoint distribution.
- **Fitts' Law timing**: movement duration is predicted from an index-of-difficulty formula
  (`a + b * log2(D/W + 1)`) with log-normal jitter, so short moves are fast and long moves take
  proportionally longer — matching the speed-accuracy trade-off of real hand movements.
- **`MovePointer` rewrite**: queries `LastCursorPos()` for the current injected cursor position
  and plays back the full SigmaDrift path with per-sample `WM_MOUSEMOVE` messages and
  gamma-distributed sleeps. If no prior position is known (first call of the session), the animation
  is skipped entirely to avoid a spurious trajectory from (0, 0).
- **`memory_injector.go` additions**: a new `cursorPosKnown` bool and
  `LastCursorPos() (x, y int, ok bool)` method so `MovePointer` can detect the first-call case.
- **Bounds clamping**: intermediate trajectory points are clamped to the game window rect before
  being packed into `lParam` to prevent corrupt bit patterns from OU drift or tremor pushing
  coordinates negative or beyond the window boundary.

### 2. Human-like timing & cursor randomness (`internal/utils/sleep.go`, 20+ action/character files)

All fixed `time.Sleep` calls in the bot lifecycle, key sequences, character selection, and
supervisor flow are replaced with `utils.Sleep()` which applies the human-like timing distributions
below. This spans `supervisor.go`, `single_supervisor.go`, `character_switch.go`,
`keyboard.go`, and all character build files.

- **`Sleep()` rewrite**: the core sleep function now draws from a **Gamma(4, 0.25)** distribution
  (mean multiplier = 1.0, right-skewed) instead of the upstream flat +/-30% uniform jitter. The
  multiplier is clamped to [0.4, 2.5] to prevent pathological extremes.
- **Session fatigue**: a progressive multiplier from 1.0 to 1.25 rises linearly over the first
  3 hours, modelling mild reaction-time slowdown in extended play sessions. `SetSessionStart()` is
  called at each new game; `sessionMu` is a `RWMutex` with `RLock` in the read path.
- **`RandGammaDurationMs`**: walk-polling and movement steps use a gamma-distributed duration
  instead of narrow uniform windows.
- **`RandLogNormal`**: idle gaps (inter-game pauses, cursor wander dwells) use a log-normal
  distribution matching empirical human idle-time data.
- **Click-position jitter**: buff casts, CTA casts, and item pickup positions receive small random
  offsets. Pickup spiral coordinates get per-step random offsets.
- **Attack / cast timing jitter**: attack sleeps in all character builds (berserk barb, warcry barb,
  whirlwind barb, paladin, assassin, both barb leveling files, blizzard/fireball/hydraorb/lightning/
  nova sorceress, hammerdin, foh, javazon, mosaic, trapsin, wind druid, druid leveling, necromancer
  leveling, amazon leveling, Smiter) have +/- jitter added to break metronomic cast intervals.

### 3. Inter-game idle behaviour (`internal/bot/single_supervisor.go`)

- **Log-normal inter-game idle**: after a game ends, a randomised pause is sampled from a log-normal
  distribution (configurable via `InterGameIdleMinMs` / `InterGameIdleMaxMs` in config, defaults
  4000–20000 ms) instead of a flat 3-second or 5-second wait. The right-skewed distribution is
  harder to distinguish from human between-game gaps.
- **Idle cursor wander (`idleCursorWander`)**: 0–4 small cursor movements to random on-screen
  positions during the inter-game pause, with a geometric-like count distribution
  (P(0)~55%, P(1)~25%, P(2)~11%, ...) to mimic human fidgeting on the character-select screen.
- **Randomised client-close waits**: the fixed 3 s / 5 s waits after game-finish or error are
  replaced with `RandRng(2500, 6000)` and `RandRng(4000, 9000)` respectively.
- **`SetSessionStart()`** is called at the top of each new game so that the fatigue multiplier
  resets properly.
- Removed the `defer runCancel()`; `runCancel()` is now called explicitly after the game loop
  exits to ensure per-game context cancellation fires at the right time.

### 4. Scheduler activation & dormant UI (`internal/bot/scheduler.go`, `internal/server/`, `internal/config/`)

- Per-supervisor scheduler activation tracking with mutex-guarded `ActivateCharacter` /
  `DeactivateCharacter` / `IsActivated` helpers.
- Scheduler is automatically activated on non-manual starts (including auto-start flow) and
  deactivated on Stop.
- **Simple schedule mode** added (default changed from `"timeSlots"` to `"simple"`): just a
  daily start and stop time (supports overnight windows e.g. 22:00–06:00). New config fields:
  `SimpleStartTime`, `SimpleStopTime`.
- New `WaitingForSchedule` supervisor status.
- HTTP API exposes `Activated`, `ScheduleSummary`, `SimpleStopTime`, `WaitingForSchedule`,
  and `ScheduledStartTime` fields in scheduler status responses; a `scheduleSummary()` helper
  produces human-readable summaries for simple/duration/time-slots modes.
- `cancelPendingStart` is called on config save to prevent stale schedule goroutines from firing
  at now-outdated times.
- `daysOfWeek` array reordered to start with Sunday and moved outside the loop.
- Dashboard CSS adds dormant and header-badge states; dashboard JS shows a compact scheduler badge,
  a dormant summary when the scheduler is enabled but not yet activated, improved countdown rendering
  with `countdown-live` elements, and a 30-second auto-refresh to keep countdowns accurate.

### 5. Supervisor manager concurrency safety (`internal/bot/manager.go`)

- A `sync.RWMutex` is added to protect the `supervisors` and `crashDetectors` maps.
- All map reads (`AvailableSupervisors`, `Stats`, `GetData`, `GetContext`) hold `RLock`;
  mutations (`Start`, `Stop`, `ReloadConfig`) hold the write lock.
- `Start` performs a double-check under write lock to prevent a race where two concurrent calls
  both pass the initial `RLock` existence check.
- `Stop` extracts references under the lock but calls `s.Stop()` and `cd.Stop()` **outside**
  the lock to avoid deadlocking with `restartFunc` (the crash-detector goroutine).
- `ReloadConfig` takes a snapshot of running supervisors and applies configs outside the lock.
- **`Runtime` preservation**: `ctx.CharacterCfg.Runtime` (compiled NIP rules, tier rules, etc.)
  is saved before the reload and restored afterwards so pickit continues to work immediately after a
  hot-reload. The upstream code had this commented out.

### 6. Bot state, stash safety & logging refactors (`internal/action/`, `internal/bot/`, `internal/context/`, `internal/game/`)

- **Per-supervisor monster-state tracking** in `action/step/attack.go`: monster state maps are
  keyed by bot name (mutex-guarded) to eliminate `UnitID` collisions between concurrent
  supervisors.
- **Stash gold slice guards** in `action/stash.go`: length checks before indexing `StashedGold`,
  safe total computation, removal of noisy debug prints.
- `maxInteractions` in `action/step/pickup_item.go` made function-local so high-attempt modes
  get extra tries and the global variable is removed.
- `bot/bot.go` `shouldReturnToTown` simplified with early returns; never returns if already in
  town or in UberTristram.
- `bot/single_supervisor.go` removes the erroneous reset of `FailedToCreateGameAttempts` in the
  modal-absent branch.
- `context/context.go` `Get()` panics on unregistered goroutines to surface misuse;
  `getGoroutineID` uses a smaller buffer and faster numeric parse.
- `game/manager.go` and `game/packet_sender.go` replace `fmt` debug prints with structured
  `slog` logging.
- `character/sorceress_leveling.go` debug messages converted to `ctx.Logger.Debug` with context
  fields.
- `bot/supervisor.go` `logGameStart` handles empty run list without panic.
- `bot/supervisor.go` removes the `disconnected`-based VK_DOWN/VK_UP workaround for character
  selection after reconnect — the code was fragile and is no longer needed.

### 7. Config additions (`internal/config/config.go`)

- `InterGameIdleMinMs` / `InterGameIdleMaxMs` fields added to `CharacterCfg.Game` — control
  the randomised idle pause between game exit and the next game creation (defaults 4000/20000 ms).
- `SaveSupervisorConfig` now calls `Validate()` **before** `yaml.Marshal` so that any field
  corrections (e.g. NovaSorceress `BossStaticThreshold`) are present in the written YAML. Upstream
  calls `Validate()` after marshalling, which means corrections are lost.

### 8. Safety guards for nil/bounds panics (`internal/run/`, `internal/action/`, `internal/town/`)

- Bounds checks added before accessing `NPC.Positions[0]` across `interaction.go`, `anya.go`,
  `quests.go`, `cave.go`, `bone_ash.go`, `jail.go`, `izual.go`, `countess.go`,
  `A1.go`.
- `action/move.go` shrine lookup: fix shadowed variable that caused the best-shrine result to
  always be `nil`; result is now stored in a scoped variable and returned after the loop.
- `action/repair.go`: remove unused import alias; call `context.Get()` directly.
- `action/vendor.go`: replace removed `botCtx` alias with `context.Get()`; adjust Jamella key
  sequence to skip `VK_DOWN`; enforce `MaxGameLength` only when it is greater than zero.

### 9. Portal/waypoint refresh guards (`internal/run/leveling_act4.go`, `leveling_act5.go`)

- After sending the Harrogath portal key sequence, wait, refresh game data, and re-query the portal;
  return an error if it is still missing (both act-4 portal locations).
- Act-5 waypoint usage guarded by existence check before calling `MoveToCoords` to prevent a nil
  dereference.

### 10. A* pathfinder fixes (`internal/pather/astar/astar.go`, `path_finder.go`, `render_map.go`)

- **Index layout corrected to row-major**: buffer index function changed from `x*height + y` to
  `y*width + x`, matching the game grid's actual row-major memory layout. The upstream column-major
  index could silently corrupt the cost/cameFrom arrays for non-square grids.
- **Stale priority-queue entries skipped**: a new guard `current.Cost > costSoFar[...]` skips nodes
  that already have a cheaper path recorded, avoiding wasted expansions and subtly incorrect paths.
- **Struct literal field names**: `data.Position{0, 1}` changed to `data.Position{X: 0, Y: 1}`
  for clarity and forward-compatibility with struct changes.
- **`GetClosestWalkablePath`**: search step changed from 4 to 1 for finer resolution; loop bounds
  use `<=` instead of `<` to include the shell boundary; perimeter test simplified from
  `math.Abs` to direct `==` comparison, removing the `math` import.
- **Lut Gholein grid fix**: the non-walkable tile at (210, 13) is now written to a local grid copy
  instead of mutating the shared `a.Grid`, preventing permanent corruption of live map data.
- **`renderMap` allocations reduced**: the path-lookup map switches from `map[string]bool`
  (`fmt.Sprintf`-keyed) to `map[data.Position]bool` (struct-keyed), eliminating per-tile string
  allocation and the `fmt` import. The redundant `draw.Draw` call is removed.

### 11. Server / HTTP hardening (`internal/server/`)

- **Template error handling**: every `ExecuteTemplate` call now checks the returned error and logs
  it with `slog.Error`. Upstream silently discards template rendering failures.
- **Method guards**: `resetMuling`, `openDroplogs`, and `resetDroplogs` endpoints now reject
  non-POST requests with 405 Method Not Allowed.
- **Telegram chat ID**: parse error is only surfaced when Telegram is actually enabled, preventing a
  spurious validation failure when the field is empty and Telegram is off.
- **Chest mutual exclusivity**: when `InteractWithChests` is checked, `InteractWithSuperChests`
  is forced off to prevent contradictory config.
- **Diablo `AttackFromDistance` form key**: was reading the wrong field
  (`gameLevelingHellRequiredFireRes`); corrected to `gameDiabloAttackFromDistance`.
- **`getIntFromForm` default-value logging**: the warning log now prints the actual
  `defaultValue` instead of a hardcoded `0`.
- **`BarbLeveling.UsePacketLearning`**: checkbox scope was incorrectly inside an inner `if`
  block; moved to the correct scope so the value is always read from the form.
- **Pickit API**: duplicate `sendJSON` call removed; `strconv.Atoi` replaces `fmt.Sscanf` for
  line-number parsing (clearer error handling).
- **Shopping wiring**: `form.Has()` replaces the custom `postedBool()` helper (which is removed),
  aligning with the standard library checkbox idiom.
- **Runewords**: `strings.TrimSuffix` replaces manual string slicing for rune name cleanup.

### 12. Miscellaneous small fixes

- **`event/listener.go`**: `rand.Intn(math.MaxInt64)` changed to `rand.Intn(math.MaxInt)` to
  avoid overflow on 32-bit or future platforms.
- **`updater/revert.go`**: `fmt.Errorf(result.Error)` replaced with `errors.New(result.Error)`
  to satisfy the `go vet` / `staticcheck` diagnostic for non-constant format strings.
- **`updater/updater.go`**: added a log message when backing up old executables so the user can see
  progress during updates.
- **`keyboard.go`**: `KeySequence` inter-key delay changed from fixed `time.Sleep(200ms)` to
  `utils.Sleep(200)` for timing humanisation.

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