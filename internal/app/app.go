// Package app wires the whole desktop pet together and owns the pet's
// behavior, movement and animation on the UI thread.
package app

import (
	"fmt"
	"image"
	"math/rand"
	"path/filepath"
	"runtime"

	"desktop-pet"
	"desktop-pet/internal/animation"
	"desktop-pet/internal/behavior"
	"desktop-pet/internal/config"
	"desktop-pet/internal/logging"
	"desktop-pet/internal/monitor"
	"desktop-pet/internal/movement"
	"desktop-pet/internal/renderer"
	"desktop-pet/internal/spritegen"
	"desktop-pet/internal/tray"
	"desktop-pet/internal/window"
)

const timerAnim = 1

// App implements window.Handler and runs the pet lifecycle.
type App struct {
	cfg      *config.Config
	win      *window.Window
	renderer *renderer.Renderer
	tray     *tray.Tray
	rng      *rand.Rand

	anims  map[animation.State]*animation.Animation
	sm     animation.StateMachine
	mover  movement.Mover
	engine *behavior.Engine

	// sprite size
	w, h int

	// physics / position
	x       float64
	y       float64
	vy      float64
	groundY float64

	// behavior
	countdown  float64
	walkTime   float64
	safetyTick float64

	// interaction
	dragging    bool
	dragOffsetX int
	dragOffsetY int

	paused bool
	hidden bool
}

// Run initializes everything and runs the window message loop.
func Run() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	window.SetProcessDPIAware()

	cfgDir, err := config.Dir()
	if err != nil {
		return err
	}
	if err := logging.Init(filepath.Join(cfgDir, "logs")); err != nil {
		fmt.Fprintln(stderr, "logging init:", err)
	}
	logging.Printf("Application started")

	cfg, err := config.Load()
	if err != nil {
		logging.Printf("config load failed: %v", err)
	}
	cfg.Validate()
	// External registry state wins.
	cfg.AutoStart = config.AutoStartEnabled()

	anims, sw, sh, err := loadAnimations(cfg.Scale)
	if err != nil {
		return fmt.Errorf("load sprites: %w", err)
	}

	a := &App{
		cfg:    cfg,
		rng:    rand.New(rand.NewSource(rand.Int63())),
		anims:  anims,
		w:      sw,
		h:      sh,
		engine: behavior.NewEngine(cfg.IdleMinSec, cfg.IdleMaxSec, nil),
	}

	win, err := window.New(a, sw, sh)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}
	a.win = win
	a.renderer = renderer.New(win)

	// Apply persisted window options.
	if !cfg.AlwaysOnTop {
		win.SetTopMost(false)
	}
	if cfg.ClickThrough {
		win.SetClickThrough(true)
	}

	// Place pet on the primary monitor work area.
	wa, err := monitor.Primary()
	if err != nil {
		return err
	}
	a.mover = movement.Mover{
		X:        float64(wa.Left + 20),
		Width:    float64(sw),
		Dir:      movement.Right,
		Speed:    cfg.MovementSpeed,
		LeftMin:  float64(wa.Left),
		RightMax: float64(wa.Right),
	}
	a.groundY = float64(wa.Bottom) - float64(sh) - 8
	a.y = a.groundY
	a.x = a.mover.X

	// Tray.
	a.tray = tray.New(win.Hwnd())
	if frame := anims[animation.Happy].Frames[0]; frame != nil {
		if err := a.tray.SetIcon(frame); err == nil {
			if err := a.tray.Add(); err != nil {
				logging.Printf("tray add failed: %v", err)
			}
		}
	}
	logging.Printf("Tray initialized")

	// Show pet and start ticking.
	a.enterIdle()
	a.renderer.Draw(a.sm.Frame(), int(a.x), int(a.y), a.opacity())
	win.Show(true)
	win.SetTimer(timerAnim, uint(1000/cfg.FPS))

	logging.Printf("Pet loaded")
	win.Run()

	// Cleanup.
	win.KillTimer(timerAnim)
	if a.tray != nil {
		a.tray.Delete()
	}
	win.Destroy()
	a.saveConfig()
	logging.Printf("Application exited")
	return nil
}

func (a *App) saveConfig() {
	a.cfg.Validate()
	if err := a.cfg.Save(); err != nil {
		logging.Printf("config save failed: %v", err)
	}
}

func (a *App) timerMs() uint { return uint(1000 / a.cfg.FPS) }

func (a *App) opacity() byte { return byte(a.cfg.Opacity*255 + 0.5) }

// ---------------------------------------------------------------------------
// Sprite loading
// ---------------------------------------------------------------------------

// loadAnimations loads all five animations from the embedded assets, falling
// back to procedural generation when the embedded files are unavailable.
func loadAnimations(scale float64) (map[animation.State]*animation.Animation, int, int, error) {
	load := func(name string, fps int, loop bool) (*animation.Animation, error) {
		return renderer.LoadAnimation(assets.Pet, "assets/pet", name, fps, loop, scale)
	}

	idle, err1 := load("idle", 3, true)
	walk, err2 := load("walk", 10, true)
	jump, err3 := load("jump", 12, false)
	sleep, err4 := load("sleep", 2, true)
	happy, err5 := load("happy", 8, false)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
		logging.Printf("embedded sprites unavailable, using generated sprites: %v %v %v %v %v", err1, err2, err3, err4, err5)
		idle = genAnim("idle", 3, true, spritegen.IdleFrames(), scale)
		walk = genAnim("walk", 10, true, spritegen.WalkFrames(), scale)
		jump = genAnim("jump", 12, false, spritegen.JumpFrames(), scale)
		sleep = genAnim("sleep", 2, true, spritegen.SleepFrames(), scale)
		happy = genAnim("happy", 8, false, spritegen.HappyFrames(), scale)
	}

	anims := map[animation.State]*animation.Animation{
		animation.Idle:      idle,
		animation.WalkLeft:  walk,
		animation.WalkRight: walk,
		animation.Jump:      jump,
		animation.Sleep:     sleep,
		animation.Happy:     happy,
	}
	f0 := idle.Frames[0]
	return anims, f0.Rect.Dx(), f0.Rect.Dy(), nil
}

func genAnim(name string, fps int, loop bool, params []spritegen.Params, scale float64) *animation.Animation {
	frames := make([]*image.RGBA, 0, len(params))
	for _, p := range params {
		img := spritegen.Draw(p)
		if scale != 1 {
			img = renderer.ScaleRGBA(img, scale)
		}
		frames = append(frames, img)
	}
	return &animation.Animation{Name: name, Frames: frames, FPS: fps, Loop: loop}
}

// ---------------------------------------------------------------------------
// Behavior
// ---------------------------------------------------------------------------

func (a *App) enterIdle() {
	a.sm.Change(animation.Idle, a.anims)
	a.countdown = a.engine.RandomIdleDuration()
	if a.rng.Intn(2) == 0 {
		a.mover.Dir = movement.Left
	} else {
		a.mover.Dir = movement.Right
	}
}

func (a *App) planNext() {
	if !a.cfg.BehaviorEnabled {
		a.enterIdle()
		return
	}
	w := behavior.Weights{
		Idle:  a.cfg.Weights["idle"],
		Walk:  a.cfg.Weights["walk"],
		Jump:  a.cfg.Weights["jump"],
		Happy: a.cfg.Weights["happy"],
		Sleep: a.cfg.Weights["sleep"],
	}
	switch a.engine.PickAction(w) {
	case behavior.Walk:
		a.sm.Change(animation.WalkLeft, a.anims)
		if a.mover.Dir == movement.Right {
			a.sm.Change(animation.WalkRight, a.anims)
		}
		a.walkTime = 2 + a.rng.Float64()*3
		a.countdown = a.walkTime
	case behavior.Jump:
		a.sm.Change(animation.Jump, a.anims)
		a.vy = a.cfg.JumpVelocity
		a.countdown = 1e9
	case behavior.Sleep:
		a.sm.Change(animation.Sleep, a.anims)
		a.countdown = 4 + a.rng.Float64()*4
	case behavior.Happy:
		a.sm.Change(animation.Happy, a.anims)
		a.countdown = 1e9
	default:
		a.enterIdle()
	}
}

// tick advances one simulation step of dt seconds.
func (a *App) tick(dt float64) {
	if a.paused || a.hidden {
		return
	}
	mustRedraw := false

	switch a.sm.State() {
	case animation.WalkLeft, animation.WalkRight:
		if a.mover.Step(dt) {
			// Hit a boundary: clamp to the monitor work area and flip.
			wa := monitor.At(int(a.x), int(a.y))
			a.mover.LeftMin = float64(wa.Left)
			a.mover.RightMax = float64(wa.Right)
			if a.mover.Dir == movement.Left {
				a.sm.Change(animation.WalkLeft, a.anims)
			} else {
				a.sm.Change(animation.WalkRight, a.anims)
			}
		}
		a.x = a.mover.X
		a.countdown -= dt
		if a.countdown <= 0 {
			a.enterIdle()
		}
		mustRedraw = true
	case animation.Jump:
		a.vy -= a.cfg.Gravity * dt
		a.y -= a.vy * dt
		if a.y >= a.groundY {
			a.y = a.groundY
			a.vy = 0
			a.enterIdle()
		}
		mustRedraw = true
	case animation.Sleep:
		a.countdown -= dt
		if a.countdown <= 0 {
			a.enterIdle()
		}
	case animation.Happy:
		if a.sm.Finished() {
			a.enterIdle()
		}
	default: // Idle
		a.countdown -= dt
		if a.countdown <= 0 {
			a.planNext()
		}
		a.checkMonitor(dt)
	}

	changed := a.sm.Tick(dt)
	if changed || mustRedraw {
		a.renderer.Draw(a.sm.Frame(), int(a.x), int(a.y), a.opacity())
	}
}

// checkMonitor runs periodically while idle. If the pet ends up outside the
// monitor it is bound to (e.g. the monitor was unplugged), it re-binds to the
// monitor under the pet.
func (a *App) checkMonitor(dt float64) {
	a.safetyTick += dt
	if a.safetyTick < 1.0 {
		return
	}
	a.safetyTick = 0
	wa := monitor.At(int(a.x)+a.w/2, int(a.y)+a.h/2)
	if int(a.x) < wa.Left || int(a.x)+a.w > wa.Right || a.y > a.groundY+1 {
		a.bindMonitor(wa)
	}
}

// bindMonitor sets the pet's ground and horizontal bounds to a monitor's work
// area and clamps the pet inside it.
func (a *App) bindMonitor(wa monitor.WorkArea) {
	a.mover.LeftMin = float64(wa.Left)
	a.mover.RightMax = float64(wa.Right)
	a.groundY = float64(wa.Bottom) - float64(a.h) - 8
	if a.y > a.groundY {
		a.y = a.groundY
	}
	if a.x < float64(wa.Left) {
		a.x = float64(wa.Left)
	}
	if maxX := float64(wa.Right) - float64(a.w); a.x > maxX {
		a.x = maxX
	}
	a.mover.X = a.x
}

// ---------------------------------------------------------------------------
// window.Handler
// ---------------------------------------------------------------------------

func (a *App) OnTimer(id uintptr) {
	if id != timerAnim {
		return
	}
	a.tick(1.0 / float64(a.cfg.FPS))
}

func (a *App) OnLeftButtonDown(x, y int) {
	if a.cfg.ClickThrough {
		return
	}
	wx, wy := a.win.Pos()
	a.dragging = true
	a.dragOffsetX = x - wx
	a.dragOffsetY = y - wy
	a.win.SetCapture()
	a.enterIdle()
}

func (a *App) OnMouseMove(x, y int) {
	if !a.dragging {
		return
	}
	a.x = float64(x - a.dragOffsetX)
	a.y = float64(y - a.dragOffsetY)
	a.win.SetPos(int(a.x), int(a.y))
}

func (a *App) OnLeftButtonUp(x, y int) {
	if !a.dragging {
		return
	}
	a.dragging = false
	a.win.ReleaseCapture()
	// Re-bind to whatever monitor the pet was dropped on.
	a.bindMonitor(monitor.At(int(a.x)+a.w/2, int(a.y)+a.h/2))
	a.enterIdle()
	a.renderer.Draw(a.sm.Frame(), int(a.x), int(a.y), a.opacity())
}

func (a *App) OnRightButtonUp(x, y int) {
	a.showMenu(x, y)
}

// OnHitTest decides whether the pixel under the cursor belongs to the pet.
func (a *App) OnHitTest(sx, sy int) bool {
	if a.cfg.ClickThrough {
		return false
	}
	wx, wy := a.win.Pos()
	frame := a.sm.Frame()
	if frame == nil {
		return true
	}
	lx := sx - wx
	ly := sy - wy
	b := frame.Bounds()
	if lx < b.Min.X || ly < b.Min.Y || lx >= b.Max.X || ly >= b.Max.Y {
		return false
	}
	return frame.RGBAAt(lx, ly).A > 10
}

func (a *App) OnUserMsg(msg uint32, wparam, lparam uintptr) {
	if msg != tray.CallbackMessage {
		return
	}
	switch lparam {
	case window.WM_RBUTTONUP:
		x, y := cursorPos()
		a.showMenu(x, y)
	case window.WM_LBUTTONUP, window.NIN_SELECT:
		if a.hidden {
			a.showPet()
		} else {
			a.hidePet()
		}
	}
}

// ---------------------------------------------------------------------------
// Menu
// ---------------------------------------------------------------------------

const (
	menuShowHide    = 1
	menuPause       = 2
	menuTopMost     = 3
	menuClickThru   = 4
	menuAutoStart   = 5
	menuFPS         = 6
	menuScale       = 7
	menuExit        = 8
	menuMoveMonitor = 100 // monitor i -> menuMoveMonitor + i
)

func (a *App) showMenu(x, y int) {
	title := "Hide Pet"
	if a.hidden {
		title = "Show Pet"
	}
	items := []tray.MenuItem{
		{ID: menuShowHide, Text: title},
		{ID: menuPause, Text: "Pause", Checked: a.paused},
		{ID: menuTopMost, Text: "Always on Top", Checked: a.cfg.AlwaysOnTop},
		{ID: menuClickThru, Text: "Click-through", Checked: a.cfg.ClickThrough},
		{ID: menuAutoStart, Text: "Start with Windows", Checked: a.cfg.AutoStart},
		{ID: menuFPS, Text: fmt.Sprintf("FPS: %d", a.cfg.FPS)},
		{ID: menuScale, Text: fmt.Sprintf("Scale: %.1fx", a.cfg.Scale)},
	}
	if monitors, err := monitor.List(); err == nil && len(monitors) > 1 {
		items = append(items, tray.MenuItem{Separator: true})
		cur := monitor.IndexAt(int(a.x)+a.w/2, int(a.y)+a.h/2, monitors)
		for _, m := range monitors {
			label := fmt.Sprintf("Move to Monitor %d", m.Index+1)
			if m.Primary {
				label += " (Primary)"
			}
			items = append(items, tray.MenuItem{
				ID:      menuMoveMonitor + m.Index,
				Text:    label,
				Checked: m.Index == cur,
			})
		}
	}
	items = append(items,
		tray.MenuItem{Separator: true},
		tray.MenuItem{ID: menuExit, Text: "Exit"},
	)
	id := tray.ShowMenu(a.win.Hwnd(), items, x, y)
	if id != 0 {
		a.handleMenu(id)
	}
}

func (a *App) handleMenu(id int) {
	if id >= menuMoveMonitor {
		a.moveToMonitor(id - menuMoveMonitor)
		return
	}
	switch id {
	case menuShowHide:
		if a.hidden {
			a.showPet()
		} else {
			a.hidePet()
		}
	case menuPause:
		a.setPaused(!a.paused)
	case menuTopMost:
		a.cfg.AlwaysOnTop = !a.cfg.AlwaysOnTop
		a.win.SetTopMost(a.cfg.AlwaysOnTop)
		a.saveConfig()
	case menuClickThru:
		a.cfg.ClickThrough = !a.cfg.ClickThrough
		a.win.SetClickThrough(a.cfg.ClickThrough)
		a.saveConfig()
	case menuAutoStart:
		a.cfg.AutoStart = !a.cfg.AutoStart
		exe, err := executablePath()
		if err == nil {
			if err := config.SetAutoStart(a.cfg.AutoStart, exe); err != nil {
				logging.Printf("auto-start failed: %v", err)
			}
		}
		a.saveConfig()
	case menuFPS:
		if a.cfg.FPS == 30 {
			a.cfg.FPS = 60
		} else {
			a.cfg.FPS = 30
		}
		a.win.KillTimer(timerAnim)
		a.win.SetTimer(timerAnim, a.timerMs())
		a.saveConfig()
	case menuScale:
		switch a.cfg.Scale {
		case 1.0:
			a.cfg.Scale = 1.5
		case 1.5:
			a.cfg.Scale = 2.0
		default:
			a.cfg.Scale = 1.0
		}
		a.reloadScale()
		a.saveConfig()
	case menuExit:
		a.exit()
	}
}

// moveToMonitor teleports the pet to the bottom-center of monitor index idx.
func (a *App) moveToMonitor(idx int) {
	monitors, err := monitor.List()
	if err != nil || idx < 0 || idx >= len(monitors) {
		return
	}
	wa := monitors[idx].WorkArea
	a.x = float64(wa.Left + (wa.Width()-a.w)/2)
	a.y = float64(wa.Bottom) - float64(a.h) - 8
	a.bindMonitor(wa)
	a.enterIdle()
	a.renderer.Draw(a.sm.Frame(), int(a.x), int(a.y), a.opacity())
	logging.Printf("Moved to monitor %d", idx+1)
}

func (a *App) hidePet() {
	a.hidden = true
	a.win.Show(false)
}

func (a *App) showPet() {
	a.hidden = false
	a.win.Show(true)
	a.renderer.Draw(a.sm.Frame(), int(a.x), int(a.y), a.opacity())
}

func (a *App) setPaused(p bool) {
	a.paused = p
	if p {
		a.win.KillTimer(timerAnim)
		logging.Printf("Paused")
	} else {
		a.win.SetTimer(timerAnim, a.timerMs())
		logging.Printf("Resumed")
	}
}

// reloadScale reloads sprites at the new scale and repositions the pet.
func (a *App) reloadScale() {
	anims, sw, sh, err := loadAnimations(a.cfg.Scale)
	if err != nil {
		logging.Printf("reload scale failed: %v", err)
		return
	}
	a.anims = anims
	a.w, a.h = sw, sh
	state := a.sm.State()
	a.sm.Change(state, a.anims)
	a.mover.Width = float64(sw)
	a.bindMonitor(monitor.At(int(a.x)+sw/2, int(a.y)+sh/2))
	a.renderer.Draw(a.sm.Frame(), int(a.x), int(a.y), a.opacity())
}

func (a *App) exit() {
	logging.Printf("Exit requested")
	a.saveConfig()
	if a.tray != nil {
		a.tray.Delete()
	}
	a.win.Destroy()
}
