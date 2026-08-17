// Command genassets generates the built-in pet sprite PNGs into assets/pet.
// Run: go run ./cmd/genassets
package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"desktop-pet/internal/spritegen"
)

var animations = []struct {
	name   string
	frames func() []spritegen.Params
}{
	{"idle", spritegen.IdleFrames},
	{"walk", spritegen.WalkFrames},
	{"jump", spritegen.JumpFrames},
	{"sleep", spritegen.SleepFrames},
	{"happy", spritegen.HappyFrames},
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	base := filepath.Join(root, "assets", "pet")
	for _, anim := range animations {
		dir := filepath.Join(base, anim.name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		params := anim.frames()
		for i, p := range params {
			img := spritegen.Draw(p)
			name := fmt.Sprintf("%02d.png", i+1)
			path := filepath.Join(dir, name)
			f, err := os.Create(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err := png.Encode(f, img); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			f.Close()
		}
		fmt.Printf("generated %s: %d frames\n", anim.name, len(params))
	}
}
