package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Config is the persisted user configuration. JSON format, plain file.
type Config struct {
	FPS             int            `json:"fps"`
	AlwaysOnTop     bool           `json:"always_on_top"`
	SoundEnabled    bool           `json:"sound_enabled"`
	AutoStart       bool           `json:"auto_start"`
	BehaviorEnabled bool           `json:"behavior_enabled"`
	ClickThrough    bool           `json:"click_through"`
	Scale           float64        `json:"scale"`
	Opacity         float64        `json:"opacity"`
	MovementSpeed   float64        `json:"movement_speed"`
	JumpVelocity    float64        `json:"jump_velocity"`
	Gravity         float64        `json:"gravity"`
	IdleMinSec      float64        `json:"idle_min_sec"`
	IdleMaxSec      float64        `json:"idle_max_sec"`
	Weights         map[string]int `json:"behavior_weights"`
}

func Default() *Config {
	return &Config{
		FPS:             30,
		AlwaysOnTop:     true,
		SoundEnabled:    false,
		AutoStart:       false,
		BehaviorEnabled: true,
		ClickThrough:    false,
		Scale:           1.0,
		Opacity:         1.0,
		MovementSpeed:   60,
		JumpVelocity:    240,
		Gravity:         700,
		IdleMinSec:      5,
		IdleMaxSec:      15,
		Weights: map[string]int{
			"idle":  30,
			"walk":  30,
			"jump":  20,
			"happy": 10,
			"sleep": 10,
		},
	}
}

// Validate clamps values into sane ranges.
func (c *Config) Validate() {
	if c.FPS != 30 && c.FPS != 60 {
		c.FPS = 30
	}
	if c.Scale <= 0 || c.Scale > 4 {
		c.Scale = 1.0
	}
	if c.Opacity < 0 {
		c.Opacity = 0
	}
	if c.Opacity > 1 {
		c.Opacity = 1
	}
	if c.MovementSpeed <= 0 {
		c.MovementSpeed = 60
	}
	if c.JumpVelocity <= 0 {
		c.JumpVelocity = 240
	}
	if c.Gravity <= 0 {
		c.Gravity = 700
	}
	if c.IdleMinSec <= 0 {
		c.IdleMinSec = 5
	}
	if c.IdleMaxSec < c.IdleMinSec {
		c.IdleMaxSec = c.IdleMinSec + 10
	}
	if c.Weights == nil {
		c.Weights = Default().Weights
	}
}

// dirOverride lets tests redirect the config directory.
var dirOverride string

// Dir returns the configuration directory (%APPDATA%\DesktopPet).
func Dir() (string, error) {
	if dirOverride != "" {
		return dirOverride, nil
	}
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		appData = home
	}
	return filepath.Join(appData, "DesktopPet"), nil
}

func path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config file. If the file is missing it returns defaults.
// If the file is corrupt it is backed up to config.bak.json and defaults are used.
func Load() (*Config, error) {
	cfg := Default()
	p, err := path()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		_ = os.WriteFile(p+".bak.json", data, 0o644)
		cfg = Default()
	}
	cfg.Validate()
	return cfg, nil
}

// Save writes the config file, creating the directory if needed.
func (c *Config) Save() error {
	p, err := path()
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// --- registry helpers (advapi32, no external dependency) --------------------

var (
	advapi         = syscall.NewLazyDLL("advapi32.dll")
	regOpenKeyEx   = advapi.NewProc("RegOpenKeyExW")
	regSetValueEx  = advapi.NewProc("RegSetValueExW")
	regDeleteValue = advapi.NewProc("RegDeleteValueW")
	regCloseKey    = advapi.NewProc("RegCloseKey")
	regGetValue    = advapi.NewProc("RegGetValueW")
)

const (
	hkeyCurrentUser = 0x80000001
	regValueSZ      = 1
	keySetValue     = 0x0002
	keyQueryValue   = 0x0001
	regNone         = 0x00000002 // RRF_RT_REG_SZ flag for RegGetValue
)

// SetAutoStart registers (or removes) the app in HKCU Run. No admin required.
func SetAutoStart(enabled bool, exePath string) error {
	var key uintptr
	sub, _ := syscall.UTF16PtrFromString(runKey)
	access := uint32(keySetValue)
	r, _, _ := regOpenKeyEx.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(sub)), 0, uintptr(access), uintptr(unsafe.Pointer(&key)))
	if r != 0 {
		return fmt.Errorf("open run key: %v", syscall.Errno(r))
	}
	defer regCloseKey.Call(key)
	name, _ := syscall.UTF16PtrFromString("DesktopPet")
	if enabled {
		value := `"` + exePath + `"`
		data, _ := syscall.UTF16FromString(value)
		r, _, _ := regSetValueEx.Call(key, uintptr(unsafe.Pointer(name)), 0, regValueSZ,
			uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)*2))
		if r != 0 {
			return fmt.Errorf("set run value: %v", syscall.Errno(r))
		}
		return nil
	}
	r, _, _ = regDeleteValue.Call(key, uintptr(unsafe.Pointer(name)))
	if r != 0 && syscall.Errno(r) != 2 { // ERROR_FILE_NOT_FOUND is fine
		return fmt.Errorf("delete run value: %v", syscall.Errno(r))
	}
	return nil
}

// AutoStartEnabled reports whether the app is registered to auto-start.
func AutoStartEnabled() bool {
	var key uintptr
	sub, _ := syscall.UTF16PtrFromString(runKey)
	r, _, _ := regOpenKeyEx.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(sub)), 0, keyQueryValue, uintptr(unsafe.Pointer(&key)))
	if r != 0 {
		return false
	}
	defer regCloseKey.Call(key)
	name, _ := syscall.UTF16PtrFromString("DesktopPet")
	var buf [512]uint16
	var size uint32 = uint32(len(buf) * 2)
	r, _, _ = regGetValue.Call(key, 0, uintptr(unsafe.Pointer(name)), regNone, 0,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	return r == 0
}
