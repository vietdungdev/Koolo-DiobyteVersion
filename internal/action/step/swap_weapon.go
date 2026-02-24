package step

import (
	"time"

	"github.com/hectorgimenez/d2go/pkg/data/skill"
	"github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/utils"
)

func SwapToMainWeapon() error {
	return swapWeapon(false)
}

func SwapToCTA() error {
	return swapWeapon(true)
}

func swapWeapon(toCTA bool) error {
	lastRun := time.Time{}
	const maxSwapAttempts = 6
	attempts := 0

	ctx := context.Get()
	ctx.SetLastStep("SwapToCTA")

	for {
		// Pause the execution if the priority is not the same as the execution priority
		ctx.PauseIfNotPriority()

		if time.Since(lastRun) < time.Millisecond*500 {
			continue
		}

		_, found := ctx.Data.PlayerUnit.Skills[skill.BattleOrders]
		if (toCTA && found) || (!toCTA && !found) {
			return nil
		}

		if attempts >= maxSwapAttempts {
			if toCTA {
				ctx.Logger.Warn("Failed to swap to CTA after max attempts, BattleOrders not found on either weapon set")
			} else {
				ctx.Logger.Warn("Failed to swap to main weapon after max attempts")
			}
			return nil
		}

		ctx.HID.PressKeyBinding(ctx.Data.KeyBindings.SwapWeapons)
		utils.PingSleep(utils.Light, 150)

		attempts++
		lastRun = time.Now()
	}
}
