package monitor

import (
	"desktop-pet/internal/window"
)

// WorkArea is a screen rectangle in virtual-screen coordinates.
type WorkArea struct {
	Left, Top, Right, Bottom int
}

func (w WorkArea) Width() int  { return w.Right - w.Left }
func (w WorkArea) Height() int { return w.Bottom - w.Top }

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
// Primary when the query fails. This is the hook for multi-monitor support.
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
