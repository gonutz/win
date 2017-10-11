package win

import (
	"errors"
	"syscall"

	"github.com/gonutz/w32"
)

type MessageCallback func(window w32.HWND, msg uint32, w, l uintptr) uintptr

// NewWindow creates a window of the given size with a border and a title bar
// with an icon, title text, minimize, maximize and close buttons (style
// WS_OVERLAPPEDWINDOW).
func NewWindow(x, y, width, height int, className string, f MessageCallback) (w32.HWND, error) {
	class := w32.WNDCLASSEX{
		WndProc:   syscall.NewCallback(f),
		Cursor:    w32.LoadCursor(0, w32.MakeIntResource(w32.IDC_ARROW)),
		ClassName: syscall.StringToUTF16Ptr(className),
		Style:     w32.CS_OWNDC, // NOTE this is needed for OpenGL
	}
	atom := w32.RegisterClassEx(&class)
	if atom == 0 {
		return 0, errors.New("win.NewWindow: RegisterClassEx failed")
	}
	window := w32.CreateWindowEx(
		0,
		syscall.StringToUTF16Ptr(className),
		nil,
		w32.WS_OVERLAPPEDWINDOW|w32.WS_VISIBLE,
		x, y, width, height,
		0, 0, 0, nil,
	)
	if window == 0 {
		return 0, errors.New("win.NewWindow: CreateWindowEx failed")
	}
	return window, nil
}

// SetIconFromExe sets the icon in the window title bar, in the taskbar and when
// using Alt-Tab to switch between applications.
// The icon is loaded from the running executable file using the given resource
// ID. This means that the icon must be embedded in the executable when building
// by using a resource file for example.
func SetIconFromExe(window w32.HWND, resourceID uint16) {
	iconHandle := w32.LoadImage(
		w32.GetModuleHandle(""),
		w32.MakeIntResource(resourceID),
		w32.IMAGE_ICON,
		0,
		0,
		w32.LR_DEFAULTSIZE|w32.LR_SHARED,
	)
	if iconHandle != 0 {
		w32.SendMessage(window, w32.WM_SETICON, w32.ICON_SMALL, uintptr(iconHandle))
		w32.SendMessage(window, w32.WM_SETICON, w32.ICON_SMALL2, uintptr(iconHandle))
		w32.SendMessage(window, w32.WM_SETICON, w32.ICON_BIG, uintptr(iconHandle))
	}
}

// IsFullscreen returns true if the window has a style different from
// WS_OVERLAPPEDWINDOW. The EnableFullscreen function will change the style to
// borderless so this reports whether that function was called on the window.
// It is not a universally valid test for any window to see if it is fullscreen.
// It is intended for use in conjunction with EnableFullscreen and
// DisableFullscreen.
func IsFullscreen(window w32.HWND) bool {
	style := w32.GetWindowLong(window, w32.GWL_STYLE)
	return style&w32.WS_OVERLAPPEDWINDOW == 0
}

// EnableFullscreen makes the window a borderless window that covers the full
// area of the monitor under the window.
// It returns the previous window placement. Store that value and use it with
// DisableFullscreen to reset the window to what it was before.
func EnableFullscreen(window w32.HWND) (windowed w32.WINDOWPLACEMENT) {
	style := w32.GetWindowLong(window, w32.GWL_STYLE)
	var monitorInfo w32.MONITORINFO
	monitor := w32.MonitorFromWindow(window, w32.MONITOR_DEFAULTTOPRIMARY)
	if w32.GetWindowPlacement(window, &windowed) &&
		w32.GetMonitorInfo(monitor, &monitorInfo) {
		w32.SetWindowLong(
			window,
			w32.GWL_STYLE,
			uint32(style & ^w32.WS_OVERLAPPEDWINDOW),
		)
		w32.SetWindowPos(
			window,
			0,
			int(monitorInfo.RcMonitor.Left),
			int(monitorInfo.RcMonitor.Top),
			int(monitorInfo.RcMonitor.Right-monitorInfo.RcMonitor.Left),
			int(monitorInfo.RcMonitor.Bottom-monitorInfo.RcMonitor.Top),
			w32.SWP_NOOWNERZORDER|w32.SWP_FRAMECHANGED,
		)
	}
	w32.ShowCursor(false)
	return
}

// DisableFullscreen makes the window have a border and the standard icons
// (style WS_OVERLAPPEDWINDOW) and places it at the position given by the window
// placement parameter.
// Use this in conjunction with IsFullscreen and EnableFullscreen to toggle a
// window's fullscreen state.
func DisableFullscreen(window w32.HWND, placement w32.WINDOWPLACEMENT) {
	style := w32.GetWindowLong(window, w32.GWL_STYLE)
	w32.SetWindowLong(
		window,
		w32.GWL_STYLE,
		uint32(style|w32.WS_OVERLAPPEDWINDOW),
	)
	w32.SetWindowPlacement(window, &placement)
	w32.SetWindowPos(window, 0, 0, 0, 0, 0,
		w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOZORDER|
			w32.SWP_NOOWNERZORDER|w32.SWP_FRAMECHANGED,
	)
	w32.ShowCursor(true)
}

// RunMainLoop starts the applications window message handling. It loops until
// the window is closed. Messages are forwarded to the handler function that was
// passed to NewWindow.
func RunMainLoop() {
	var msg w32.MSG
	for w32.GetMessage(&msg, 0, 0, 0) != 0 {
		w32.TranslateMessage(&msg)
		w32.DispatchMessage(&msg)
	}
}

// RunMainGameLoop starts the applications window message handling. It loops
// until the window is closed. Messages are forwarded to the handler function
// that was passed to NewWindow.
// In contrast to RunMainLoop, RunMainGameLoop calls the given function whenever
// there are now messages to be handled at the moment. You can use this like a
// classical DOS era endless loop to run any real-time logic in between
// messages.
// Tip: if you do not want the game to use all your CPU, do some kind of
// blocking operation in the function you pass. A simple time.Sleep(0) will do
// the trick.
func RunMainGameLoop(f func()) {
	var msg w32.MSG
	w32.PeekMessage(&msg, 0, 0, 0, w32.PM_NOREMOVE)
	for msg.Message != w32.WM_QUIT {
		if w32.PeekMessage(&msg, 0, 0, 0, w32.PM_REMOVE) {
			w32.TranslateMessage(&msg)
			w32.DispatchMessage(&msg)
		} else {
			f()
		}
	}
}

// CloseWindow sends a WM_CLOSE event to the given window.
func CloseWindow(window w32.HWND) {
	w32.SendMessage(window, w32.WM_CLOSE, 0, 0)
}

// HideConsoleWindow hides the associated console window if it was created
// because the ldflag H=windowsgui was not provided when building.
func HideConsoleWindow() {
	console := w32.GetConsoleWindow()
	if console == 0 {
		return // no console attached
	}
	// If this application is the process that created the console window, then
	// this program was not compiled with the -H=windowsgui flag and on start-up
	// it created a console along with the main application window. In this case
	// hide the console window.
	// See
	// http://stackoverflow.com/questions/9009333/how-to-check-if-the-program-is-run-from-a-console
	// and thanks to
	// https://github.com/hajimehoshi
	// for the tip.
	_, consoleProcID := w32.GetWindowThreadProcessId(console)
	if w32.GetCurrentProcessId() == consoleProcID {
		w32.ShowWindowAsync(console, w32.SW_HIDE)
	}
}
