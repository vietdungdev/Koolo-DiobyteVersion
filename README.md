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

### 13. Telegram & Discord startup resilience (`cmd/koolo/main.go`, `internal/remote/telegram/constructor.go`)

- **Non-fatal bot initialization**: Discord and Telegram bot initialization errors are now logged as
  warnings instead of crashing the application. The bot continues running without the messaging
  service that failed. This fixes the reported issue where a TCP connection reset during Telegram
  API startup would crash the entire application.
- **Telegram retry with exponential backoff**: the Telegram constructor retries up to 3 times with
  exponential backoff (2 s → 4 s → 8 s) before giving up, handling transient network errors
  (TCP resets, DNS failures, timeouts) gracefully.

### 14. Discord API migration (`internal/remote/discord/discord_event_handler.go`)

- **`MessageSend.File` → `MessageSend.Files`**: migrated from the deprecated singular `File` field
  to the `Files` slice (discordgo v0.29.0). Both `sendItemScreenshot` and `sendScreenshot` are
  updated. This prevents future breakage when the deprecated field is removed.

### 15. Andariel search fix (`internal/run/andariel.go`, `internal/character/paladin_leveling.go`)

- **Centralized `searchForAndariel()` method**: added 6 progressively deeper search positions in
  Andariel's chamber (Y coordinates 9560 → 9520), with the final position approaching Andariel's
  throne directly. Before killing, the run now calls `searchForAndariel()` which moves through each
  position and checks for the boss via `data.Monsters.FindOne()`. This fixes the reported issue
  where the bot would stand at the chamber entrance unable to find Andariel because she was deeper
  in the room. The fix benefits **all 20+ character classes** since it lives in the run layer, not
  per-character.
- **Paladin leveling directed search**: `KillAndariel()` in `paladin_leveling.go` now uses
  `action.MoveToCoords` targeting Andariel's known throne coordinates `(22548, 9520)` instead of
  `RandomMovement()` when the boss is not found. Random movement could move the character
  further away from the spawn; directed movement converges on her location.

### 16. Bot idle state / stuck-in-town fix (`internal/action/move.go`)

- **Town return detection with timeout**: when the bot is unexpectedly teleported to town during
  field movement (e.g., accidental TP click during combat), a `townReturnDetectedAt` timer starts.
  After 5 seconds stuck in town, the bot proactively calls `UsePortalInTown()` to return to the
  field. Previously, the `MoveTo` loop would spin indefinitely with `Sleep(100)` calls, appearing
  as if the bot was standing still doing nothing.
- **`!pathFound` branch missing `continue`**: when the player was in town and
  `UsePortalInTown()` succeeded in the `!pathFound` branch, execution fell through into path
  step computation with a nil `path`, causing `path[pathStep]` to index out of range. The panic
  was silently swallowed by the goroutine recovery handler, killing the run goroutine while
  health/refresh goroutines kept ticking — producing the observed "idle" state. A `continue` is
  now added after the successful portal call so the loop restarts and recalculates the path
  from the post-portal position.
- **Defensive `pathStep >= 0` guard**: before indexing into `path` in the non-town movement
  code path, a bounds guard now prevents out-of-range access if `path` is empty, mirroring
  the existing guard already present in the town branch.

### 17. Attack repositioning improvement (`internal/action/step/attack.go`)

- **Increased reposition attempts**: `repositionAttempts` threshold raised from `>= 1` to `>= 3`,
  giving the bot more chances to angle around obstacles before giving up on a monster. This prevents
  premature target abandonment in tight corridors.

### 18. Quest item stash fix (`internal/action/stash.go`)

- **Operator precedence bug fix**: the `shouldStashIt()` function had an erroneous
  `|| i.Name == "HoradricStaff"` that bypassed all stash-exclusion logic due to Go's operator
  precedence. The condition was removed so quest items are now correctly evaluated against the
  standard stash rules.

### 19. Updater repository migration (`internal/updater/`, `internal/server/templates/config.gohtml`)

- **All updater URLs point to this fork**: the updater now clones, fetches, and checks against
  `Diobyte/Koolo-DiobyteVersion` instead of `kwader2k/koolo`:
  - `repo.go`: clone URL updated
  - `git.go`: `ensureUpstreamRemote()` — upstream URL, expected URL, and contains-check all updated
  - `pr.go`: `upstreamOwner` → `"Diobyte"`, `upstreamRepo` → `"Koolo-DiobyteVersion"` (affects
    GitHub API calls for PR listing, commit fetching, cherry-pick)
- **GUI text updated**: all `kwader2k/koolo` references in the web UI (how-it-works panel,
  version fallback message, update status text, PR "Open" links) now show
  `Diobyte/Koolo-DiobyteVersion`.
- **Go module paths unchanged**: `github.com/hectorgimenez/koolo` references in `GOGARBLE` and
  `-ldflags` are Go import paths matching `go.mod` and are intentionally left as-is.

### 20. Infinite loop / bot freeze fixes (`internal/action/step/`, `internal/context/`)

Three unbounded loops that could cause the bot to stand still indefinitely have been fixed:

- **`swapWeapon()` — `swap_weapon.go`**: the `for {}` loop had **no max attempts**. If
  `SwapToCTA()` was called but no CTA existed (e.g., `UseSwapForBuffs` enabled without a CTA
  equipped), the bot would spin forever pressing the weapon swap key every 500 ms. Fixed with
  `maxSwapAttempts = 6`; after exhausting attempts, logs a warning and returns gracefully.
- **`WaitForGameToLoad()` — `context.go`**: the `for LoadingScreen` loop had **no timeout**. If
  the loading screen flag got stuck (frozen game client, network issue), the bot would block
  forever. This is called from 6+ critical code paths. Fixed with a **30-second deadline**; after
  timeout, logs a warning and proceeds.
- **`OpenPortal()` — `open_portal.go`**: the `for {}` loop had **no max attempts**. If the portal
  object never appeared (laggy server, area restriction, game state desync), the loop would retry
  every 1 second indefinitely. Fixed with `maxPortalAttempts = 10`; after exhausting attempts,
  returns an error that propagates up for proper game restart.

### 21. Blizzard Sorceress threat assessment & positioning rewrite (`internal/character/blizzard_sorceress.go`)

- **Weighted threat scoring system**: the old per-monster `needsRepositioning()` binary check is
  replaced with `assessThreat()` which iterates all nearby enemies once and computes a weighted
  composite score factoring in:
  - Proximity (inverse distance from danger zone)
  - Elite multiplier (×3.0 for elite mobs)
  - Dangerous aura detection (Fanaticism, Might, Conviction, Holy Fire/Shock/Freeze — ×2.5)
  - Aggregate monster count threshold (≥3 nearby enemies triggers regardless of individual scores)
- **Threat-level thresholds**: Low (3.0), Medium (8.0), High (15.0) drive graduated responses
  instead of the previous single-distance binary trigger.
- **Dynamic reposition cooldown**: `getRepositionCooldown()` interpolates between 200 ms and
  1200 ms based on HP percentage (60% weight) and normalized threat score (40% weight).
  Lower HP or higher threat → shorter cooldown → more frequent repositioning. Replaces the
  fixed 1-second cooldown.
- **Centroid-based escape vector**: `findSafePosition()` now computes the threat centroid
  (weighted center of all nearby enemies) and escapes **away from the pack** rather than away
  from a single arbitrary monster, preventing teleports into flanking groups.
- **Two-phase directional search**: safe position candidates are first searched in a
  180° cone centered on the escape vector (10° increments), falling back to a full 360°
  circle only if no walkable positions exist in the cone.
- **Enhanced scoring formula**: candidate positions are now scored with additional factors:
  centroid distance (+2.5 weight), nearby monster count penalty (−3.0 per mob), nearby
  elite/aura threat score penalty (−1.5), and a wall-blocked line-of-sight bonus (+1.0 per
  blocked dangerous monster) that favours positions where terrain shields from elites.
- **Cooldown-phase repositioning**: during Blizzard cooldown (when previously the bot
  would always primary-attack), the bot now repositions if the threat assessment warrants
  it, using the same safe-position logic. Falls back to primary attack only when the area
  is safe.
- **`stutterStepStaticField()` helper**: Static Field usage on bosses (Izual, Diablo, Baal)
  is extracted into a shared helper that alternates between close-range Static Field casts
  and teleport-retreats to safe Blizzard distance. Includes HP-gated abort (< 50% HP),
  boss-alive checks, and Hell difficulty threshold detection (stops Static at ~35% boss HP
  since it cannot reduce below ~33%).
- **Nil-safety on boss lookups**: `KillIzual`, `KillBaal`, and `KillDiablo` now check the
  `found` return from `FindOne` before using the monster.
- **Removed dead code**: unused `needsRepositioning()`, stale commented-out `KillMephisto`
  block, and orphaned `minMonsterDistance()` helper are removed.

### 22. Clear area/room abort on area change & unreachable mob skip (`internal/action/clear_area.go`, `internal/action/clear_level.go`)

- **Area-change detection**: both `ClearThroughPath()` and `clearRoom()` now record the starting
  area at entry and check `ctx.Data.PlayerUnit.Area` each iteration. If the player has been
  unexpectedly moved to a different area (death, chicken, waypoint), the function aborts
  immediately with a descriptive error instead of spinning against stale map data.
- **Skip unreachable monsters**: `clearRoom()` tracks a `skippedMonsters` set. When the
  pathfinder cannot find a path to a target monster, the monster's `UnitID` is blacklisted
  for the remainder of that room clearing pass, preventing infinite retry loops on monsters
  stuck behind walls or in unreachable geometry.

### 23. Shared stash support for Horadric Cube & quest items (`internal/action/horadric_cube.go`, `internal/action/cube_recipes.go`, `internal/run/cube.go`, `internal/run/quests.go`)

When `StashToShared` is enabled, the Horadric Cube can legitimately end up in a shared stash tab.
Previously, multiple code paths only searched personal stash and inventory for the cube, causing
false "cube not found" errors and unnecessary re-fetch attempts.

- **`ensureCubeIsOpen()`**: `Find("HoradricCube", ...)` now includes `item.LocationSharedStash`.
  The existing `SwitchStashTab(cube.Location.Page + 1)` formula already handles both personal
  (Page 0 → Tab 1) and shared pages (Page N → Tab N+1) correctly.
- **`CubeAddItems()`**: the old `requiresPersonalStash` if/else guard is replaced with a
  location-based `switch` that navigates to whatever tab the item currently occupies, handling
  personal stash, shared stash, and DLC tabs uniformly.
- **`cube_recipes.go`**: material search locations extended to include `LocationSharedStash`.
- **`run/cube.go` and `run/quests.go`**: Horadric Cube presence checks now search shared stash,
  preventing the bot from re-running the cube quest when it already has one stashed.

> Note: In D2R, the Horadric Cube can be placed in the shared stash. Other quest components
> (Staff of Kings, Amulet of the Viper, etc.) are server-side blocked from shared stash tabs,
> so the non-cube quest item path is defensive but does not affect gameplay.

---

> For a file-level diff against upstream run:
> ```sh
> git fetch upstream && git diff --stat upstream/main
> ```

## Notes

- The game must still be set to **English**. 1280x720 windowed mode and LOD 1.13c are required as usual.
- All other documentation (installation, usage, pickit rules, etc.) is unchanged from upstream — refer to
  the [Diobyte/Koolo-DiobyteVersion](https://github.com/Diobyte/Koolo-DiobyteVersion) README or the
  original [kwader2k/koolo](https://github.com/kwader2k/koolo) README for full details.

---

*This README covers only the modifications made in this fork. See the upstream project for the full Koolo documentation.*