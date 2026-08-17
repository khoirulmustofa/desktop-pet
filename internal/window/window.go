package window

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"
)

// Handler receives window events on the UI thread.
type Handler interface {
	// OnTimer fires from SetTimer. timerID identifies the timer.
	OnTimer(timerID uintptr)
	// OnMouseMove reports cursor screen coordinates.
	OnMouseMove(x, y int)
	// OnLeftButtonDown/Up report cursor screen coordinates.
	OnLeftButtonDown(x, y int)
	OnLeftButtonUp(x, y int)
	// OnRightButtonUp reports cursor screen coordinates.
	OnRightButtonUp(x, y int)
	// OnHitTest returns true when the pixel is part of the pet (opaque).
	OnHitTest(x, y int) bool
	// OnUserMsg receives app-defined messages (WM_APP+...), e.g. tray callbacks.
	OnUserMsg(msg uint32, wparam, lparam uintptr)
}

type Window struct {
	hwnd    uintptr
	handler Handler
	width   int
	height  int
}

// current is the live window instance used by the shared WndProc.
var current *Window

const windowClassName = "DesktopPetWindow"

func init() {
	runtime.LockOSThread()
}

// New registers the window class and creates a borderless, transparent,
// always-on-top, tool (no taskbar button) window. It stays hidden until Show.
func New(handler Handler, width, height int) (*Window, error) {
	hInstance, err := getModuleHandle()
	if err != nil {
		return nil, err
	}

	wndProc := syscall.NewCallback(wndProcFn)

	className, _ := syscall.UTF16PtrFromString(windowClassName)

	wcx := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         0,
		LpfnWndProc:   wndProc,
		CbClsExtra:    0,
		CbWndExtra:    0,
		HInstance:     hInstance,
		HIcon:         0,
		HCursor:       loadCursor(),
		HbrBackground: 0,
		LpszMenuName:  nil,
		LpszClassName: className,
		HIconSm:       0,
	}

	atom, _, _ := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wcx)))
	if atom == 0 {
		return nil, errors.New("RegisterClassExW failed")
	}

	exStyle := uint32(WS_EX_LAYERED | WS_EX_TOPMOST | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE)
	style := uint32(WS_POPUP)

	hwnd, _, _ := pCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(atom),
		uintptr(unsafe.Pointer(className)),
		uintptr(style),
		0, 0,
		uintptr(width), uintptr(height),
		0, // parent
		0, // menu
		hInstance,
		0, // param
	)
	if hwnd == 0 {
		return nil, errors.New("CreateWindowExW failed")
	}

	w := &Window{hwnd: hwnd, handler: handler, width: width, height: height}
	current = w
	return w, nil
}

func getModuleHandle() (uintptr, error) {
	// GetModuleHandleW from kernel32 via syscall.
	dll := syscall.NewLazyDLL("kernel32.dll")
	proc := dll.NewProc("GetModuleHandleW")
	r, _, e := proc.Call(0)
	if r == 0 {
		return 0, e
	}
	return r, nil
}

func (w *Window) Hwnd() uintptr { return w.hwnd }

func (w *Window) Show(visible bool) {
	if visible {
		showWindow(w.hwnd, SW_SHOWNOACTIVATE)
	} else {
		showWindow(w.hwnd, SW_HIDE)
	}
}

// SetPos moves the window to screen coordinates (x, y) at its current size.
func (w *Window) SetPos(x, y int) {
	setWindowPos(w.hwnd, 0, x, y, 0, 0, SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE)
}

// Pos returns the window's screen position (top-left corner).
func (w *Window) Pos() (int, int) {
	r := getWindowRect(w.hwnd)
	return int(r.Left), int(r.Top)
}

func (w *Window) Size() (int, int) { return w.width, w.height }

// SetTopMost toggles always-on-top z-order.
func (w *Window) SetTopMost(top bool) {
	insert := uintptr(HWND_TOPMOST)
	if !top {
		insert = uintptr(HWND_NOTOPMOST)
	}
	setWindowPos(w.hwnd, insert, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE)
}

// SetClickThrough toggles WS_EX_TRANSPARENT so all mouse input passes through.
func (w *Window) SetClickThrough(on bool) {
	style := getWindowLongPtr(w.hwnd, GWL_EXSTYLE)
	if on {
		style |= WS_EX_TRANSPARENT
	} else {
		style &^= WS_EX_TRANSPARENT
	}
	setWindowLongPtr(w.hwnd, GWL_EXSTYLE, style)
	setWindowPos(w.hwnd, 0, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER|SWP_FRAMECHANGED|SWP_NOACTIVATE)
}

// SetTimer starts a periodic WM_TIMER.
func (w *Window) SetTimer(id uintptr, ms uint) {
	setTimer(w.hwnd, id, ms)
}

func (w *Window) KillTimer(id uintptr) {
	killTimer(w.hwnd, id)
}

// CreatePopupMenu creates a popup menu handle for the tray / right-click menu.
func (w *Window) CreatePopupMenu() uintptr {
	r, _, _ := pCreatePopupMenu.Call()
	return r
}

func (w *Window) DestroyMenu(menu uintptr) {
	pDestroyMenu.Call(menu)
}

// AppendMenuString adds a string menu item. flags may combine MF_* constants.
func (w *Window) AppendMenuString(menu uintptr, flags uint32, id uintptr, text string) {
	s, _ := syscall.UTF16PtrFromString(text)
	pAppendMenuW.Call(menu, uintptr(flags), id, uintptr(unsafe.Pointer(s)))
}

// TrackPopupMenuAt shows a menu at screen coordinates and returns the selected id
// (TPM_RETURNCMD) or 0 when dismissed.
func (w *Window) TrackPopupMenuAt(menu uintptr, x, y int) uintptr {
	r, _, _ := pTrackPopupMenu.Call(
		menu,
		uintptr(TPM_RIGHTBUTTON|TPM_RETURNCMD|TPM_LEFTALIGN|TPM_BOTTOMALIGN),
		uintptr(x), uintptr(y), 0, w.hwnd, 0)
	return r
}

// Run runs the Windows message loop until the window is destroyed.
func (w *Window) Run() {
	var msg MSG
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 { // WM_QUIT
			return
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// Destroy posts WM_CLOSE/terminates and destroys the window.
func (w *Window) Destroy() {
	if w.hwnd != 0 {
		pDestroyWindow.Call(w.hwnd)
		w.hwnd = 0
	}
}

// PostQuit posts WM_QUIT to end the message loop.
func (w *Window) PostQuit() {
	pPostQuitMessage.Call(0)
}

// SetCapture captures the mouse so drags outside the window keep arriving.
func (w *Window) SetCapture() {
	pSetCapture.Call(w.hwnd)
}

func (w *Window) ReleaseCapture() {
	pReleaseCapture.Call(0)
}

// UpdateLayered blits the given 32bpp premultiplied BGRA buffer to the window
// using per-pixel alpha, also repositioning the window to (x, y).
func (w *Window) UpdateLayered(x, y int, width, height int, pixels []byte, opacity byte) {
	if len(pixels) < width*height*4 {
		return
	}

	hdcScreen := getDC(0)
	if hdcScreen == 0 {
		return
	}
	hdcMem := createCompatibleDC(hdcScreen)
	if hdcMem == 0 {
		releaseDC(0, hdcScreen)
		return
	}

	var bmi BITMAPINFO
	bmi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bmi.BmiHeader))
	bmi.BmiHeader.BiWidth = int32(width)
	bmi.BmiHeader.BiHeight = -int32(height) // top-down
	bmi.BmiHeader.BiPlanes = 1
	bmi.BmiHeader.BiBitCount = 32
	bmi.BmiHeader.BiCompression = BI_RGB

	var bits unsafe.Pointer
	hbm, _, _ := pCreateDIBSection.Call(
		hdcMem,
		uintptr(unsafe.Pointer(&bmi)),
		DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&bits)),
		0, 0)
	if hbm == 0 || bits == nil {
		deleteDC(hdcMem)
		releaseDC(0, hdcScreen)
		return
	}
	old := selectObject(hdcMem, hbm)

	dst := unsafe.Slice((*byte)(bits), width*height*4)
	copy(dst, pixels)

	blend := BLENDFUNCTION{
		BlendOp:             AC_SRC_OVER,
		SourceConstantAlpha: opacity,
		AlphaFormat:         AC_SRC_ALPHA,
	}
	var ptDst POINT
	ptDst.X = int32(x)
	ptDst.Y = int32(y)
	var sz POINT
	sz.X = int32(width)
	sz.Y = int32(height)
	var ptSrc POINT

	pUpdateLayeredWindow.Call(
		w.hwnd,
		hdcScreen,
		uintptr(unsafe.Pointer(&ptDst)),
		uintptr(unsafe.Pointer(&sz)),
		hdcMem,
		uintptr(unsafe.Pointer(&ptSrc)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		ULW_ALPHA,
	)

	selectObject(hdcMem, old)
	deleteObject(hbm)
	deleteDC(hdcMem)
	releaseDC(0, hdcScreen)
}

// wndProcFn dispatches window messages to the live window's handler.
func wndProcFn(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	w := current
	switch msg {
	case WM_DESTROY:
		pPostQuitMessage.Call(0)
		return 0
	case WM_CLOSE:
		pPostQuitMessage.Call(0)
		return 0
	case WM_ERASEBKGND:
		return 1 // layered window: never erase
	case WM_TIMER:
		if w != nil && w.handler != nil {
			w.handler.OnTimer(wparam)
		}
		return 0
	case WM_NCHITTEST:
		if w != nil && w.handler != nil {
			x := int(int16(lparam & 0xFFFF))
			y := int(int16((lparam >> 16) & 0xFFFF))
			if w.handler.OnHitTest(x, y) {
				return HTCLIENT
			}
			return uintptr(HTTRANSPARENT)
		}
	case WM_LBUTTONDOWN:
		if w != nil && w.handler != nil {
			x, y := getCursorPos()
			w.handler.OnLeftButtonDown(x, y)
		}
		return 0
	case WM_MOUSEMOVE:
		if w != nil && w.handler != nil {
			x, y := getCursorPos()
			w.handler.OnMouseMove(x, y)
		}
		return 0
	case WM_LBUTTONUP:
		if w != nil && w.handler != nil {
			x, y := getCursorPos()
			w.handler.OnLeftButtonUp(x, y)
		}
		return 0
	case WM_RBUTTONUP:
		if w != nil && w.handler != nil {
			x, y := getCursorPos()
			w.handler.OnRightButtonUp(x, y)
		}
		return 0
	case WM_SETCURSOR:
		// Keep the arrow cursor.
		return 0
	}
	if msg >= WM_APP && w != nil && w.handler != nil {
		w.handler.OnUserMsg(msg, wparam, lparam)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wparam, lparam)
	return r
}
