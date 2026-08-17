package monitor

import (
	"math"
	"sort"

	"desktop-pet/internal/window"
)

// WorkArea is a screen rectangle in virtual-screen coordinates.
type WorkArea struct {
	Left, Top, Right, Bottom int
}

func (w WorkArea) Width() int  { return w.Right - w.Left }
func (w WorkArea) Height() int { return w.Bottom - w.Top }

// Info describes one monitor.
type Info struct {
	Index    int
	WorkArea WorkArea
	Primary  bool
}

// Primary returns the primary monitor work area (taskbar excluded).
func Primary() (WorkArea, error) {
	r, err := window.PrimaryWorkArea()
	if err != nil {
		return WorkArea{}, err
	}
	return WorkArea{
		Left:   int(r.Left),
		Top:    int(r.Top),
		Right:  int(r.Right),
		Bottom: int(r.Bottom),
	}, nil
}

// At returns the work area of the monitor containing (x, y). Falls back to
// Primary when the query fails.
func At(x, y int) WorkArea {
	mi, _ := window.MonitorInfoAt(x, y)
	if mi.RcWork.Right == 0 && mi.RcWork.Bottom == 0 {
		if p, err := Primary(); err == nil {
			return p
		}
	}
	return WorkArea{
		Left:   int(mi.RcWork.Left),
		Top:    int(mi.RcWork.Top),
		Right:  int(mi.RcWork.Right),
		Bottom: int(mi.RcWork.Bottom),
	}
}

// List enumerates every monitor, sorted left-to-right so "Monitor 1/2/…"
// stays stable. The primary monitor is marked.
func List() ([]Info, error) {
	raw, err := window.EnumDisplays()
	if err != nil {
		return nil, err
	}
	sort.Slice(raw, func(i, j int) bool {
		if raw[i].RcWork.Left != raw[j].RcWork.Left {
			return raw[i].RcWork.Left < raw[j].RcWork.Left
		}
		return raw[i].RcWork.Top < raw[j].RcWork.Top
	})
	infos := make([]Info, 0, len(raw))
	for i, mi := range raw {
		infos = append(infos, Info{
			Index: i,
			WorkArea: WorkArea{
				Left:   int(mi.RcWork.Left),
				Top:    int(mi.RcWork.Top),
				Right:  int(mi.RcWork.Right),
				Bottom: int(mi.RcWork.Bottom),
			},
			Primary: mi.DwFlags&window.MONITORINFOF_PRIMARY != 0,
		})
	}
	return infos, nil
}

// IndexAt returns the index (into List()) of the monitor containing (x, y).
// Returns -1 when monitors is empty. Points outside every monitor resolve to
// the nearest monitor by center distance.
func IndexAt(x, y int, monitors []Info) int {
	for _, m := range monitors {
		wa := m.WorkArea
		if x >= wa.Left && x < wa.Right && y >= wa.Top && y < wa.Bottom {
			return m.Index
		}
	}
	best := -1
	bestD := math.MaxInt
	for _, m := range monitors {
		wa := m.WorkArea
		cx := wa.Left + wa.Width()/2
		cy := wa.Top + wa.Height()/2
		dx := x - cx
		dy := y - cy
		d := dx*dx + dy*dy
		if d < bestD {
			bestD = d
			best = m.Index
		}
	}
	return best
}
