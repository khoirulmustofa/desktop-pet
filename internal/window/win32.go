package window

// Package window contains the thin Win32 layer used by the whole app:
// window management, GDI blitting, tray and monitor helpers. All syscall
// bindings live here so they can be audited in one place.

import (
	"syscall"
	"unsafe"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	WS_POPUP   = 0x80000000
	WS_VISIBLE = 0x10000000

	WS_EX_LAYERED     = 0x00080000
	WS_EX_TOPMOST     = 0x00000008
	WS_EX_TOOLWINDOW  = 0x00000080
	WS_EX_TRANSPARENT = 0x00000020
	WS_EX_NOACTIVATE  = 0x08000000

	GWL_EXSTYLE = -20

	SW_HIDE           = 0
	SW_SHOWNORMAL     = 1
	SW_SHOWNOACTIVATE = 4
	SW_SHOW           = 5

	HWND_TOPMOST   = ^uintptr(0) // -1
	HWND_NOTOPMOST = ^uintptr(1) // -2

	SWP_NOSIZE       = 0x0001
	SWP_NOMOVE       = 0x0002
	SWP_NOZORDER     = 0x0004
	SWP_NOACTIVATE   = 0x0010
	SWP_FRAMECHANGED = 0x0020
	SWP_SHOWWINDOW   = 0x0040

	WM_NULL        = 0x0000
	WM_CREATE      = 0x0001
	WM_DESTROY     = 0x0002
	WM_MOVE        = 0x0003
	WM_SIZE        = 0x0005
	WM_CLOSE       = 0x0010
	WM_QUIT        = 0x0012
	WM_PAINT       = 0x000F
	WM_ERASEBKGND  = 0x0014
	WM_TIMER       = 0x0113
	WM_COMMAND     = 0x0111
	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_MOUSEMOVE   = 0x0200
	WM_RBUTTONUP   = 0x0205
	WM_NCHITTEST   = 0x0084
	WM_SETCURSOR   = 0x0020
	WM_APP         = 0x8000

	HTCLIENT      = 1
	HTTRANSPARENT = ^uintptr(0) // -1

	MK_LBUTTON = 0x0001

	SM_CXSCREEN  = 0
	SM_CYSCREEN  = 1
	SM_CMONITORS = 80

	SPI_GETWORKAREA = 0x0030

	IDC_ARROW = 32512

	// Popup menu
	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800
	MF_CHECKED   = 0x00000008
	MF_UNCHECKED = 0x00000000
	MF_GRAYED    = 0x00000001

	TPM_RIGHTBUTTON = 0x0002
	TPM_RETURNCMD   = 0x0100
	TPM_LEFTALIGN   = 0x0000
	TPM_BOTTOMALIGN = 0x0020

	// Layered window
	ULW_ALPHA    = 2
	AC_SRC_OVER  = 0
	AC_SRC_ALPHA = 1

	// DIB
	BI_RGB         = 0
	DIB_RGB_COLORS = 0
	SRCCOPY        = 0x00CC0020

	// Tray
	NIM_ADD            = 0
	NIM_MODIFY         = 1
	NIM_DELETE         = 2
	NIM_SETVERSION     = 4
	NIF_MESSAGE        = 0x00000001
	NIF_ICON           = 0x00000002
	NIF_TIP            = 0x00000004
	NOTIFYICON_VERSION = 4
	NIN_SELECT         = 0x0400
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type POINT struct {
	X int32
	Y int32
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

type BLENDFUNCTION struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type NOTIFYICONDATAW struct {
	CbSize           uint32
	Hwnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         GUID
	HBalloonIcon     uintptr
}

type ICONINFO struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

const (
	MONITOR_DEFAULTTONEAREST = 2
	MONITORINFOF_PRIMARY     = 1
)

type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

// ---------------------------------------------------------------------------
// LazyDLL helpers
// ---------------------------------------------------------------------------

type lazydll struct {
	name string
	dll  *syscall.LazyDLL
}

func (d *lazydll) proc(name string) *syscall.LazyProc { return d.dll.NewProc(name) }

var (
	user32  = &lazydll{"user32.dll", syscall.NewLazyDLL("user32.dll")}
	gdi32   = &lazydll{"gdi32.dll", syscall.NewLazyDLL("gdi32.dll")}
	shell32 = &lazydll{"shell32.dll", syscall.NewLazyDLL("shell32.dll")}
)

var (
	pRegisterClassExW      = user32.proc("RegisterClassExW")
	pCreateWindowExW       = user32.proc("CreateWindowExW")
	pDefWindowProcW        = user32.proc("DefWindowProcW")
	pGetMessageW           = user32.proc("GetMessageW")
	pTranslateMessage      = user32.proc("TranslateMessage")
	pDispatchMessageW      = user32.proc("DispatchMessageW")
	pPostQuitMessage       = user32.proc("PostQuitMessage")
	pDestroyWindow         = user32.proc("DestroyWindow")
	pShowWindow            = user32.proc("ShowWindow")
	pSetWindowPos          = user32.proc("SetWindowPos")
	pSetTimer              = user32.proc("SetTimer")
	pKillTimer             = user32.proc("KillTimer")
	pGetDC                 = user32.proc("GetDC")
	pReleaseDC             = user32.proc("ReleaseDC")
	pSystemParametersInfoW = user32.proc("SystemParametersInfoW")
	pLoadCursorW           = user32.proc("LoadCursorW")
	pSetCapture            = user32.proc("SetCapture")
	pReleaseCapture        = user32.proc("ReleaseCapture")
	pGetCursorPos          = user32.proc("GetCursorPos")
	pGetWindowRect         = user32.proc("GetWindowRect")
	pSetWindowLongPtrW     = user32.proc("SetWindowLongPtrW")
	pGetWindowLongPtrW     = user32.proc("GetWindowLongPtrW")
	pCreatePopupMenu       = user32.proc("CreatePopupMenu")
	pDestroyMenu           = user32.proc("DestroyMenu")
	pAppendMenuW           = user32.proc("AppendMenuW")
	pTrackPopupMenu        = user32.proc("TrackPopupMenu")
	pGetMonitorInfoW       = user32.proc("GetMonitorInfoW")
	pUpdateLayeredWindow   = user32.proc("UpdateLayeredWindow")
	pCreateIconIndirect    = user32.proc("CreateIconIndirect")
	pDestroyIcon           = user32.proc("DestroyIcon")

	pCreateCompatibleDC = gdi32.proc("CreateCompatibleDC")
	pCreateDIBSection   = gdi32.proc("CreateDIBSection")
	pSelectObject       = gdi32.proc("SelectObject")
	pDeleteObject       = gdi32.proc("DeleteObject")
	pDeleteDC           = gdi32.proc("DeleteDC")
	pCreateBitmap       = gdi32.proc("CreateBitmap")

	pShellNotifyIconW = shell32.proc("Shell_NotifyIconW")
)

// ---------------------------------------------------------------------------
// Calls
// ---------------------------------------------------------------------------

// loadCursor loads the standard arrow cursor. MAKEINTRESOURCE passes the
// integer resource id directly (high word zero -> treated as resource id).
func loadCursor() uintptr {
	r, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	return r
}

func systemParametersInfo(action uint32, uiParam uint32, pvParam unsafe.Pointer, fWinIni uint32) {
	pSystemParametersInfoW.Call(uintptr(action), uintptr(uiParam), uintptr(pvParam), uintptr(fWinIni))
}

func getDC(hwnd uintptr) uintptr {
	r, _, _ := pGetDC.Call(hwnd)
	return r
}

func releaseDC(hwnd, hdc uintptr) {
	pReleaseDC.Call(hwnd, hdc)
}

func getWindowRect(hwnd uintptr) RECT {
	var r RECT
	pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return r
}

func setWindowLongPtr(hwnd uintptr, index int, value uintptr) uintptr {
	r, _, _ := pSetWindowLongPtrW.Call(hwnd, uintptr(index), value)
	return r
}

func getWindowLongPtr(hwnd uintptr, index int) uintptr {
	r, _, _ := pGetWindowLongPtrW.Call(hwnd, uintptr(index))
	return r
}

func setWindowPos(hwnd uintptr, hwndInsertAfter uintptr, x, y, cx, cy int, flags uint32) {
	pSetWindowPos.Call(hwnd, hwndInsertAfter, uintptr(x), uintptr(y), uintptr(cx), uintptr(cy), uintptr(flags))
}

func showWindow(hwnd uintptr, cmd int) {
	pShowWindow.Call(hwnd, uintptr(cmd))
}

func setTimer(hwnd uintptr, id uintptr, elapse uint) {
	pSetTimer.Call(hwnd, id, uintptr(elapse), 0)
}

func killTimer(hwnd uintptr, id uintptr) {
	pKillTimer.Call(hwnd, id)
}

func getCursorPos() (x, y int) {
	var p POINT
	r, _, _ := pGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		return 0, 0
	}
	return int(p.X), int(p.Y)
}

// GetCursorPos returns the cursor screen position.
func GetCursorPos() (int, int) { return getCursorPos() }

// SetProcessDPIAware makes GetSystemMetrics / coordinates physical-pixel based.
func SetProcessDPIAware() {
	syscall.NewLazyDLL("user32.dll").NewProc("SetProcessDPIAware").Call()
}

func createCompatibleDC(hdc uintptr) uintptr {
	r, _, _ := pCreateCompatibleDC.Call(hdc)
	return r
}

func deleteDC(hdc uintptr) {
	pDeleteDC.Call(hdc)
}

func selectObject(hdc, obj uintptr) uintptr {
	r, _, _ := pSelectObject.Call(hdc, obj)
	return r
}

func deleteObject(obj uintptr) {
	pDeleteObject.Call(obj)
}

// MonitorInfo returns the work area of the primary monitor via SPI_GETWORKAREA.
func PrimaryWorkArea() (RECT, error) {
	var r RECT
	pSystemParametersInfoW.Call(SPI_GETWORKAREA, 0, uintptr(unsafe.Pointer(&r)), 0)
	return r, nil
}

// MonitorInfoAt returns the monitor info containing the given point.
func MonitorInfoAt(x, y int) (MONITORINFO, uintptr) {
	var mi MONITORINFO
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	var pt POINT
	pt.X = int32(x)
	pt.Y = int32(y)
	dll := syscall.NewLazyDLL("user32.dll")
	mfp := dll.NewProc("MonitorFromPoint")
	h, _, _ := mfp.Call(uintptr(unsafe.Pointer(&pt)), MONITOR_DEFAULTTONEAREST)
	if h != 0 {
		pGetMonitorInfoW.Call(h, uintptr(unsafe.Pointer(&mi)))
	}
	return mi, h
}
