package game

import (
	"math"
	"math/rand"
	"time"

	"github.com/lxn/win"
)

const (
	RightButton MouseButton = win.MK_RBUTTON
	LeftButton  MouseButton = win.MK_LBUTTON

	ShiftKey ModifierKey = win.VK_SHIFT
	CtrlKey  ModifierKey = win.VK_CONTROL
)

type MouseButton uint
type ModifierKey byte

const pointerReleaseDelay = 150 * time.Millisecond

// MovePointer moves the mouse to the requested position using a SigmaDrift
// trajectory — a biomechanically grounded path generated from the last known
// cursor position to (x, y). x and y are relative to the game window: the
// top-left corner of the client area is (0, 0).
func (hid *HID) MovePointer(x, y int) {
	hid.gr.updateWindowPositionData()
	absX := hid.gr.WindowLeftX + x
	absY := hid.gr.WindowTopY + y

	startX, startY, ok := hid.gi.LastCursorPos()
	if !ok {
		// No prior cursor position known (first move of the session).
		// Skip animation entirely: a trajectory from an unknown origin would
		// be meaningless and would add a spurious ~50 ms sleep.
		hid.gi.CursorPos(absX, absY) //nolint:errcheck
		lParam := calculateLparam(absX, absY)
		win.SendMessage(hid.gr.HWND, win.WM_NCHITTEST, 0, lParam)
		win.SendMessage(hid.gr.HWND, win.WM_SETCURSOR, 0x000105A8, 0x2010001)
		win.PostMessage(hid.gr.HWND, win.WM_MOUSEMOVE, 0, lParam)
		return
	}

	// Generate a SigmaDrift trajectory in absolute screen coordinates.
	path := sigmaDriftGenerate(float64(startX), float64(startY), float64(absX), float64(absY), defaultSDConfig)

	// Clamp bounds for intermediate points to prevent negative or overflow values
	// in calculateLparam. OU drift, tremor, and SDN noise can push points slightly
	// outside the window rect; clamping avoids corrupt lParam bit patterns.
	maxX := hid.gr.WindowLeftX + hid.gr.GameAreaSizeX
	maxY := hid.gr.WindowTopY + hid.gr.GameAreaSizeY

	// Play back intermediate points: update injected cursor + send WM_MOUSEMOVE,
	// sleeping between samples as specified by the gamma-distributed timestamps.
	for i := 0; i+1 < len(path); i++ {
		pt := path[i]
		px := int(math.Round(pt.x))
		py := int(math.Round(pt.y))
		if px < 0 {
			px = 0
		} else if px > maxX {
			px = maxX
		}
		if py < 0 {
			py = 0
		} else if py > maxY {
			py = maxY
		}
		hid.gi.CursorPos(px, py) //nolint:errcheck
		win.PostMessage(hid.gr.HWND, win.WM_MOUSEMOVE, 0, calculateLparam(px, py))
		if dt := path[i+1].t - pt.t; dt > 0 {
			time.Sleep(time.Duration(dt) * time.Millisecond)
		}
	}

	// Micro-correction (12 % probability): simulate the brief re-aim that
	// humans make after landing near the target. Move 2–5 px off-target,
	// pause for a short dwell time, then land exactly on the target with the
	// finalise block below. This breaks the otherwise perfectly-sharp
	// endpoint distribution visible in event-log analysis.
	if rand.Float64() < 0.12 {
		ox := rand.Intn(11) - 5 // -5 to +5 px
		oy := rand.Intn(11) - 5
		mx := absX + ox
		my := absY + oy
		if mx < 0 {
			mx = 0
		}
		if my < 0 {
			my = 0
		}
		hid.gi.CursorPos(mx, my) //nolint:errcheck
		win.PostMessage(hid.gr.HWND, win.WM_MOUSEMOVE, 0, calculateLparam(mx, my))
		time.Sleep(time.Duration(rand.Intn(40)+15) * time.Millisecond)
	}

	// Finalise at the exact target using the original full message sequence.
	hid.gi.CursorPos(absX, absY) //nolint:errcheck
	lParam := calculateLparam(absX, absY)
	win.SendMessage(hid.gr.HWND, win.WM_NCHITTEST, 0, lParam)
	win.SendMessage(hid.gr.HWND, win.WM_SETCURSOR, 0x000105A8, 0x2010001)
	win.PostMessage(hid.gr.HWND, win.WM_MOUSEMOVE, 0, lParam)
}

// Click just does a single mouse click at current pointer position
func (hid *HID) Click(btn MouseButton, x, y int) {
	hid.MovePointer(x, y)
	x = hid.gr.WindowLeftX + x
	y = hid.gr.WindowTopY + y

	lParam := calculateLparam(x, y)
	buttonDown := uint32(win.WM_LBUTTONDOWN)
	buttonUp := uint32(win.WM_LBUTTONUP)
	if btn == RightButton {
		buttonDown = win.WM_RBUTTONDOWN
		buttonUp = win.WM_RBUTTONUP
	}

	win.SendMessage(hid.gr.HWND, buttonDown, 1, lParam)
	sleepTime := rand.Intn(keyPressMaxTime-keyPressMinTime) + keyPressMinTime
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	win.SendMessage(hid.gr.HWND, buttonUp, 1, lParam)
}

func (hid *HID) ClickWithModifier(btn MouseButton, x, y int, modifier ModifierKey) {
	hid.gi.OverrideGetKeyState(byte(modifier))
	hid.Click(btn, x, y)
	hid.gi.RestoreGetKeyState()
}

func calculateLparam(x, y int) uintptr {
	return uintptr(y<<16 | x)
}
