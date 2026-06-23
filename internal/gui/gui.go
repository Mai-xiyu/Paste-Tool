package gui

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Mai-xiyu/Paste-Tool/internal/config"
	"github.com/Mai-xiyu/Paste-Tool/internal/core"
	"github.com/Mai-xiyu/Paste-Tool/internal/metadata"
	"github.com/Mai-xiyu/Paste-Tool/internal/platform"
	"github.com/Mai-xiyu/Paste-Tool/internal/update"
	"golang.design/x/hotkey"
)

type controller struct {
	app     fyne.App
	window  fyne.Window
	cfg     config.Config
	cfgPath string
	driver  platform.Driver

	status *widget.Label
	diag   *widget.Label

	hotkeyMu   sync.Mutex
	hotkey     *hotkey.Hotkey
	hotkeyDone chan struct{}

	pasting atomic.Bool
}

func Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	fyneApp := app.NewWithID(metadata.AppID)
	fyneApp.SetIcon(theme.ContentPasteIcon())

	cfg, path, err := config.LoadDefault()
	if err != nil {
		return err
	}
	ctrl := &controller{
		app:     fyneApp,
		cfg:     cfg,
		cfgPath: path,
		driver:  platform.NewDriver(),
	}
	ctrl.window = ctrl.buildWindow()
	ctrl.setupTray()
	ctrl.refreshDiagnostics()
	if err := ctrl.registerHotkey(); err != nil {
		ctrl.setStatus("Hotkey unavailable: " + err.Error())
		fyneApp.SendNotification(&fyne.Notification{
			Title:   metadata.Name,
			Content: "Hotkey unavailable: " + err.Error(),
		})
	} else {
		ctrl.setStatus("Ready: " + ctrl.cfg.HotkeyString())
	}

	ctrl.window.Hide()
	fyneApp.Run()
	ctrl.unregisterHotkey()
	return nil
}

func (c *controller) buildWindow() fyne.Window {
	w := c.app.NewWindow(metadata.Name)
	w.Resize(fyne.NewSize(520, 420))
	w.SetCloseIntercept(func() { w.Hide() })

	hotkeyEntry := widget.NewEntry()
	hotkeyEntry.SetText(c.cfg.HotkeyString())
	startDelayEntry := widget.NewEntry()
	startDelayEntry.SetText(strconv.Itoa(c.cfg.Paste.StartDelayMS))
	interKeyEntry := widget.NewEntry()
	interKeyEntry.SetText(strconv.Itoa(c.cfg.Paste.InterKeyDelayMS))
	batchSizeEntry := widget.NewEntry()
	batchSizeEntry.SetText(strconv.Itoa(c.cfg.Paste.BatchSize))
	batchPauseEntry := widget.NewEntry()
	batchPauseEntry.SetText(strconv.Itoa(c.cfg.Paste.BatchPauseMS))

	c.status = widget.NewLabel("")
	c.diag = widget.NewLabel("")
	c.diag.Wrapping = fyne.TextWrapWord

	form := widget.NewForm(
		widget.NewFormItem("Hotkey", hotkeyEntry),
		widget.NewFormItem("Start delay (ms)", startDelayEntry),
		widget.NewFormItem("Inter-key delay (ms)", interKeyEntry),
		widget.NewFormItem("Batch size", batchSizeEntry),
		widget.NewFormItem("Batch pause (ms)", batchPauseEntry),
	)

	saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		next := c.cfg
		if err := next.Set("hotkey", hotkeyEntry.Text); err != nil {
			dialog.ShowError(err, w)
			return
		}
		for key, entry := range map[string]*widget.Entry{
			"paste.start_delay_ms":     startDelayEntry,
			"paste.inter_key_delay_ms": interKeyEntry,
			"paste.batch_size":         batchSizeEntry,
			"paste.batch_pause_ms":     batchPauseEntry,
		} {
			if err := next.Set(key, entry.Text); err != nil {
				dialog.ShowError(err, w)
				return
			}
		}
		if err := config.Save(c.cfgPath, next); err != nil {
			dialog.ShowError(err, w)
			return
		}
		c.cfg = next
		if err := c.registerHotkey(); err != nil {
			c.setStatus("Saved, but hotkey registration failed: " + err.Error())
			dialog.ShowError(err, w)
			return
		}
		c.setStatus("Saved: " + c.cfg.HotkeyString())
	})

	buttons := container.NewHBox(
		saveBtn,
		widget.NewButtonWithIcon("Paste", theme.ContentPasteIcon(), c.startPasteFromClipboard),
		widget.NewButtonWithIcon("Update", theme.ViewRefreshIcon(), func() { c.checkUpdate(w) }),
		widget.NewButtonWithIcon("Repository", theme.HomeIcon(), c.openRepository),
	)

	w.SetContent(container.NewVBox(
		widget.NewLabelWithStyle(metadata.Name+" "+metadata.Version, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form,
		buttons,
		widget.NewSeparator(),
		c.status,
		c.diag,
	))
	return w
}

func (c *controller) setupTray() {
	desk, ok := c.app.(desktop.App)
	if !ok {
		c.window.Show()
		return
	}
	menu := fyne.NewMenu(metadata.Name,
		fyne.NewMenuItem("Settings", func() {
			c.window.Show()
			c.window.RequestFocus()
		}),
		fyne.NewMenuItem("Paste Now", c.startPasteFromClipboard),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Check Update", func() { c.checkUpdate(c.window) }),
		fyne.NewMenuItem("Download Portable", func() { c.downloadLatest("portable") }),
		fyne.NewMenuItem("Download Installer", func() { c.downloadLatest("installer") }),
		fyne.NewMenuItem("Repository", c.openRepository),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", c.app.Quit),
	)
	desk.SetSystemTrayIcon(theme.ContentPasteIcon())
	desk.SetSystemTrayMenu(menu)
	desk.SetSystemTrayWindow(c.window)
}

func (c *controller) registerHotkey() error {
	mods, key, err := parseHotkey(c.cfg.Hotkey)
	if err != nil {
		return err
	}
	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		return err
	}

	c.hotkeyMu.Lock()
	if c.hotkeyDone != nil {
		close(c.hotkeyDone)
	}
	if c.hotkey != nil {
		_ = c.hotkey.Unregister()
	}
	done := make(chan struct{})
	c.hotkey = hk
	c.hotkeyDone = done
	c.hotkeyMu.Unlock()

	go c.listenHotkey(hk, done)
	return nil
}

func (c *controller) unregisterHotkey() {
	c.hotkeyMu.Lock()
	defer c.hotkeyMu.Unlock()
	if c.hotkeyDone != nil {
		close(c.hotkeyDone)
		c.hotkeyDone = nil
	}
	if c.hotkey != nil {
		_ = c.hotkey.Unregister()
		c.hotkey = nil
	}
}

func (c *controller) listenHotkey(hk *hotkey.Hotkey, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case <-hk.Keydown():
			c.startPasteFromClipboard()
		}
	}
}

func (c *controller) startPasteFromClipboard() {
	if !c.pasting.CompareAndSwap(false, true) {
		return
	}
	text := c.app.Clipboard().Content()
	if text == "" {
		c.pasting.Store(false)
		c.setStatus("Clipboard is empty")
		_ = c.driver.NotifyError()
		return
	}

	c.unregisterHotkey()
	c.setStatus("Pasting...")
	go func() {
		defer c.pasting.Store(false)
		defer func() {
			if err := c.registerHotkey(); err != nil {
				c.setStatus("Paste finished; hotkey registration failed: " + err.Error())
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		err := core.PasteText(ctx, text, core.Options{
			StartDelay:    c.cfg.Paste.StartDelay(),
			InterKeyDelay: c.cfg.Paste.InterKeyDelay(),
			BatchSize:     c.cfg.Paste.BatchSize,
			BatchPause:    c.cfg.Paste.BatchPause(),
		}, c.driver)
		if err != nil {
			c.setStatus("Paste failed: " + err.Error())
			return
		}
		c.setStatus("Paste finished")
	}()
}

func (c *controller) checkUpdate(parent fyne.Window) {
	c.setStatus("Checking update...")
	go func() {
		client := update.NewClient(c.cfg.Update.Repository)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		release, err := client.Latest(ctx)
		if err != nil {
			c.setStatus("Update check failed: " + err.Error())
			fyne.Do(func() { dialog.ShowError(err, parent) })
			return
		}
		if update.HasUpdate(metadata.Version, release) {
			msg := fmt.Sprintf("Update available: %s -> %s", metadata.Version, release.TagName)
			c.setStatus(msg)
			fyne.Do(func() { dialog.ShowInformation("Update", msg, parent) })
			return
		}
		msg := fmt.Sprintf("Current version %s is up to date", metadata.Version)
		c.setStatus(msg)
		fyne.Do(func() { dialog.ShowInformation("Update", msg, parent) })
	}()
}

func (c *controller) downloadLatest(kind string) {
	c.setStatus("Downloading " + kind + "...")
	go func() {
		client := update.NewClient(c.cfg.Update.Repository)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		release, err := client.Latest(ctx)
		if err != nil {
			c.setStatus("Download failed: " + err.Error())
			return
		}
		asset, err := update.SelectAsset(release, kind, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			c.setStatus("Download failed: " + err.Error())
			return
		}
		path, err := client.Download(ctx, asset, "")
		if err != nil {
			c.setStatus("Download failed: " + err.Error())
			return
		}
		c.setStatus("Downloaded: " + path)
		c.app.SendNotification(&fyne.Notification{Title: metadata.Name, Content: "Downloaded: " + path})
	}()
}

func (c *controller) openRepository() {
	u, err := url.Parse(metadata.Repository)
	if err != nil {
		c.setStatus("Invalid repository URL: " + err.Error())
		return
	}
	if err := c.app.OpenURL(u); err != nil {
		c.setStatus("Open repository failed: " + err.Error())
	}
}

func (c *controller) refreshDiagnostics() {
	lines := []string{"Config: " + c.cfgPath, "Driver: " + c.driver.Name()}
	for _, check := range c.driver.Check(context.Background()) {
		lines = append(lines, fmt.Sprintf("[%s] %s: %s", check.Status, check.Name, check.Detail))
	}
	if c.diag != nil {
		c.diag.SetText(strings.Join(lines, "\n"))
	}
}

func (c *controller) setStatus(text string) {
	if c.status == nil {
		return
	}
	fyne.Do(func() {
		c.status.SetText(text)
	})
}

func parseHotkey(cfg config.HotkeyConfig) ([]hotkey.Modifier, hotkey.Key, error) {
	mods := make([]hotkey.Modifier, 0, len(cfg.Modifiers))
	for _, mod := range cfg.Modifiers {
		parsed, ok := hotkeyModifierByName(mod)
		if ok {
			mods = append(mods, parsed)
		}
	}
	if len(mods) == 0 {
		return nil, 0, errors.New("hotkey requires at least one modifier")
	}
	key, err := hotkeyKeyByName(cfg.Key)
	if err != nil {
		return nil, 0, err
	}
	return mods, key, nil
}
