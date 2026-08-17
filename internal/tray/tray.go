// Package tray wraps Shell_NotifyIcon and the popup menus.
package tray

import (
	"image"
	"syscall"
	"unsafe"

	"desktop-pet/internal/window"
)

// Callback message the tray sends to the window. Routed via window.OnUserMsg.
const CallbackMessage = window.WM_APP + 1

// Tray manages the system tray icon.
type Tray struct {
	hwnd uintptr
	uid  uint32
	nid  window.NOTIFYICONDATAW
	icon uintptr
}

func New(hwnd uintptr) *Tray {
	return &Tray{hwnd: hwnd, uid: 1}
}

// SetIcon builds an HICON from a sprite frame and stores it for NIM_ADD/MODIFY.
func (t *Tray) SetIcon(img *image.RGBA) error {
	icon, err := iconFromRGBA(img, 32)
	if err != nil {
		return err
	}
	if t.icon != 0 {
		destroyIcon(t.icon)
	}
	t.icon = icon
	return nil
}

// Add registers the tray icon. Call after SetIcon.
func (t *Tray) Add() error {
	t.nid = window.NOTIFYICONDATAW{
		Hwnd:             t.hwnd,
		UID:              t.uid,
		UFlags:           window.NIF_MESSAGE | window.NIF_ICON | window.NIF_TIP,
		UCallbackMessage: CallbackMessage,
		HIcon:            t.icon,
	}
	title, _ := syscall.UTF16FromString("Desktop Pet")
	copy(t.nid.SzTip[:], title)
	t.nid.CbSize = uint32(unsafe.Sizeof(t.nid))
	return t.notify(window.NIM_ADD)
}

// Modify updates the tray icon (used after SetIcon changes).
func (t *Tray) Modify() error {
	t.nid.HIcon = t.icon
	t.nid.CbSize = uint32(unsafe.Sizeof(t.nid))
	return t.notify(window.NIM_MODIFY)
}

// Delete removes the tray icon.
func (t *Tray) Delete() {
	t.nid.CbSize = uint32(unsafe.Sizeof(t.nid))
	t.notify(window.NIM_DELETE)
	if t.icon != 0 {
		destroyIcon(t.icon)
		t.icon = 0
	}
}

func (t *Tray) notify(action int) error {
	r, _, _ := pShellNotifyIconW.Call(uintptr(action), uintptr(unsafe.Pointer(&t.nid)))
	if r == 0 {
		return syscall.Errno(1)
	}
	return nil
}

var pShellNotifyIconW = syscall.NewLazyDLL("shell32.dll").NewProc("Shell_NotifyIconW")

func destroyIcon(icon uintptr) {
	syscall.NewLazyDLL("user32.dll").NewProc("DestroyIcon").Call(icon)
}

// iconFromRGBA creates a square HICON from an RGBA image.
func iconFromRGBA(img *image.RGBA, size int) (uintptr, error) {
	if size == 0 {
		size = 32
	}
	b := img.Bounds()
	src := img
	if b.Dx() != size || b.Dy() != size {
		src = resizeRGBA(img, size)
	}

	hdc := createCompatibleDC(0)
	if hdc == 0 {
		return 0, syscall.Errno(1)
	}
	defer deleteDC(hdc)

	// 32bpp color DIB with premultiplied BGRA.
	var bmi window.BITMAPINFO
	bmi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bmi.BmiHeader))
	bmi.BmiHeader.BiWidth = int32(size)
	bmi.BmiHeader.BiHeight = -int32(size)
	bmi.BmiHeader.BiPlanes = 1
	bmi.BmiHeader.BiBitCount = 32
	bmi.BmiHeader.BiCompression = window.BI_RGB
	var colorBits unsafe.Pointer
	colorHbm, _, _ := pCreateDIBSection.Call(hdc, uintptr(unsafe.Pointer(&bmi)), window.DIB_RGB_COLORS, uintptr(unsafe.Pointer(&colorBits)), 0, 0)
	if colorHbm == 0 || colorBits == nil {
		return 0, syscall.Errno(1)
	}
	buf := unsafe.Slice((*byte)(colorBits), size*size*4)
	premultipliedBGRA(src, buf)

	// 1bpp mask DIB (all 0 -> fully visible).
	var mbmi window.BITMAPINFO
	mbmi.BmiHeader.BiSize = uint32(unsafe.Sizeof(mbmi.BmiHeader))
	mbmi.BmiHeader.BiWidth = int32(size)
	mbmi.BmiHeader.BiHeight = -int32(size)
	mbmi.BmiHeader.BiPlanes = 1
	mbmi.BmiHeader.BiBitCount = 1
	mbmi.BmiHeader.BiCompression = window.BI_RGB
	var maskBits unsafe.Pointer
	maskHbm, _, _ := pCreateDIBSection.Call(hdc, uintptr(unsafe.Pointer(&mbmi)), window.DIB_RGB_COLORS, uintptr(unsafe.Pointer(&maskBits)), 0, 0)
	if maskHbm == 0 {
		pDeleteObject.Call(colorHbm)
		return 0, syscall.Errno(1)
	}
	maskLen := (size*size + 31) / 32 * 4
	mbuf := unsafe.Slice((*byte)(maskBits), maskLen)
	for i := range mbuf {
		mbuf[i] = 0
	}

	ii := window.ICONINFO{FIcon: 1, HbmMask: maskHbm, HbmColor: colorHbm}
	icon, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	pDeleteObject.Call(colorHbm)
	pDeleteObject.Call(maskHbm)
	if icon == 0 {
		return 0, syscall.Errno(1)
	}
	return icon, nil
}

func resizeRGBA(img *image.RGBA, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	sw := float64(img.Bounds().Dx())
	sh := float64(img.Bounds().Dy())
	for y := 0; y < size; y++ {
		sy := int(float64(y) * sh / float64(size))
		for x := 0; x < size; x++ {
			sx := int(float64(x) * sw / float64(size))
			dst.Set(x, y, img.RGBAAt(sx, sy))
		}
	}
	return dst
}

func premultipliedBGRA(src *image.RGBA, dst []byte) {
	di := 0
	for i := 0; i < len(src.Pix); i += 4 {
		r := src.Pix[i]
		g := src.Pix[i+1]
		b := src.Pix[i+2]
		a := src.Pix[i+3]
		if a == 0 {
			dst[di], dst[di+1], dst[di+2], dst[di+3] = 0, 0, 0, 0
		} else {
			ar := int(a)
			dst[di] = byte(int(b) * ar / 255)
			dst[di+1] = byte(int(g) * ar / 255)
			dst[di+2] = byte(int(r) * ar / 255)
			dst[di+3] = a
		}
		di += 4
	}
}

var (
	pCreateDIBSection   = syscall.NewLazyDLL("gdi32.dll").NewProc("CreateDIBSection")
	pCreateIconIndirect = syscall.NewLazyDLL("user32.dll").NewProc("CreateIconIndirect")
	pDeleteObject       = syscall.NewLazyDLL("gdi32.dll").NewProc("DeleteObject")
)

func createCompatibleDC(hdc uintptr) uintptr {
	r, _, _ := syscall.NewLazyDLL("gdi32.dll").NewProc("CreateCompatibleDC").Call(hdc)
	return r
}

func deleteDC(hdc uintptr) {
	syscall.NewLazyDLL("gdi32.dll").NewProc("DeleteDC").Call(hdc)
}

// ---------------------------------------------------------------------------
// Popup menu
// ---------------------------------------------------------------------------

// MenuItem describes one line of a popup menu.
type MenuItem struct {
	ID        int
	Text      string
	Checked   bool
	Separator bool
}

// ShowMenu builds and shows a popup menu at screen (x, y). Returns the
// selected item ID or 0 when dismissed.
func ShowMenu(hwnd uintptr, items []MenuItem, x, y int) int {
	menu, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("CreatePopupMenu").Call()
	if menu == 0 {
		return 0
	}
	defer syscall.NewLazyDLL("user32.dll").NewProc("DestroyMenu").Call(menu)

	appendW := syscall.NewLazyDLL("user32.dll").NewProc("AppendMenuW")
	track := syscall.NewLazyDLL("user32.dll").NewProc("TrackPopupMenu")

	for _, it := range items {
		if it.Separator {
			appendW.Call(menu, window.MF_SEPARATOR, 0, 0)
			continue
		}
		flags := uintptr(window.MF_STRING)
		if it.Checked {
			flags |= window.MF_CHECKED
		}
		text, _ := syscall.UTF16PtrFromString(it.Text)
		appendW.Call(menu, flags, uintptr(it.ID), uintptr(unsafe.Pointer(text)))
	}

	r, _, _ := track.Call(
		menu,
		uintptr(window.TPM_RIGHTBUTTON|window.TPM_RETURNCMD|window.TPM_LEFTALIGN|window.TPM_BOTTOMALIGN),
		uintptr(x), uintptr(y), 0, hwnd, 0)
	return int(r)
}
