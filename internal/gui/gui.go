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
	"github.com/Mai-xiyu/Paste-Tool/internal/i18n"
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
	tr      i18n.Translator

	status *widget.Label
	diag   *widget.Label

	hotkeyMu         sync.Mutex
	hotkey           *hotkey.Hotkey
	hotkeyDone       chan struct{}
	registeredHotkey string

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
		tr:      i18n.New(cfg.UI.Language),
	}
	ctrl.window = ctrl.buildWindow()
	ctrl.setupTray()
	ctrl.refreshDiagnostics()
	if err := ctrl.registerHotkey(); err != nil {
		ctrl.setStatus(ctrl.tr.T(i18n.StatusHotkeyUnavailable, err.Error()))
		fyneApp.SendNotification(&fyne.Notification{
			Title:   metadata.Name,
			Content: ctrl.tr.T(i18n.StatusHotkeyUnavailable, err.Error()),
		})
	} else {
		ctrl.setStatus(ctrl.tr.T(i18n.StatusReady, ctrl.cfg.HotkeyString()))
	}

	ctrl.window.Hide()
	fyneApp.Run()
	ctrl.unregisterHotkey()
	return nil
}

func (c *controller) buildWindow() fyne.Window {
	w := c.app.NewWindow(c.tr.T(i18n.AppTitle))
	w.Resize(fyne.NewSize(520, 420))
	w.SetCloseIntercept(func() { w.Hide() })
	w.SetContent(c.buildContent(w))
	return w
}

func (c *controller) buildContent(w fyne.Window) fyne.CanvasObject {
	hotkeyEntry := widget.NewEntry()
	hotkeyEntry.SetText(c.cfg.HotkeyString())
	languageSelect := widget.NewSelect([]string{"auto", "zh-CN", "en"}, nil)
	languageSelect.SetSelected(c.cfg.UI.Language)
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
		widget.NewFormItem(c.tr.T(i18n.LabelHotkey), hotkeyEntry),
		widget.NewFormItem(c.tr.T(i18n.LabelLanguage), languageSelect),
		widget.NewFormItem(c.tr.T(i18n.LabelStartDelay), startDelayEntry),
		widget.NewFormItem(c.tr.T(i18n.LabelInterKeyDelay), interKeyEntry),
		widget.NewFormItem(c.tr.T(i18n.LabelBatchSize), batchSizeEntry),
		widget.NewFormItem(c.tr.T(i18n.LabelBatchPause), batchPauseEntry),
	)

	saveBtn := widget.NewButtonWithIcon(c.tr.T(i18n.ButtonSave), theme.DocumentSaveIcon(), func() {
		previous := c.cfg
		next := c.cfg
		if err := next.Set("hotkey", hotkeyEntry.Text); err != nil {
			dialog.ShowError(err, w)
			return
		}
		if err := next.Set("ui.language", languageSelect.Selected); err != nil {
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
		c.cfg = next
		c.tr = i18n.New(c.cfg.UI.Language)
		if err := c.registerHotkey(); err != nil {
			c.cfg = previous
			c.tr = i18n.New(c.cfg.UI.Language)
			w.SetTitle(c.tr.T(i18n.AppTitle))
			w.SetContent(c.buildContent(w))
			c.setupTray()
			c.refreshDiagnostics()
			c.setStatus(c.tr.T(i18n.StatusHotkeyUnavailable, err.Error()))
			dialog.ShowError(err, w)
			return
		}
		if err := config.Save(c.cfgPath, next); err != nil {
			c.cfg = previous
			c.tr = i18n.New(c.cfg.UI.Language)
			restoreErr := c.registerHotkey()
			w.SetTitle(c.tr.T(i18n.AppTitle))
			w.SetContent(c.buildContent(w))
			c.setupTray()
			c.refreshDiagnostics()
			if restoreErr != nil {
				c.setStatus(c.tr.T(i18n.StatusHotkeyUnavailable, restoreErr.Error()))
			}
			dialog.ShowError(err, w)
			return
		}
		w.SetTitle(c.tr.T(i18n.AppTitle))
		w.SetContent(c.buildContent(w))
		c.setupTray()
		c.refreshDiagnostics()
		c.setStatus(c.tr.T(i18n.StatusSaved, c.cfg.HotkeyString()))
	})

	buttons := container.NewHBox(
		saveBtn,
		widget.NewButtonWithIcon(c.tr.T(i18n.ButtonPaste), theme.ContentPasteIcon(), c.startPasteFromClipboard),
		widget.NewButtonWithIcon(c.tr.T(i18n.ButtonUpdate), theme.ViewRefreshIcon(), func() { c.checkUpdate(w) }),
		widget.NewButtonWithIcon(c.tr.T(i18n.ButtonRepository), theme.HomeIcon(), c.openRepository),
	)

	return container.NewVBox(
		widget.NewLabelWithStyle(metadata.Name+" "+metadata.Version, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form,
		buttons,
		widget.NewSeparator(),
		c.status,
		c.diag,
	)
}

func (c *controller) setupTray() {
	desk, ok := c.app.(desktop.App)
	if !ok {
		c.window.Show()
		return
	}
	quit := fyne.NewMenuItem(c.tr.T(i18n.MenuQuit), c.app.Quit)
	quit.IsQuit = true
	menu := fyne.NewMenu(c.tr.T(i18n.AppTitle),
		fyne.NewMenuItem(c.tr.T(i18n.MenuSettings), func() {
			c.window.Show()
			c.window.RequestFocus()
		}),
		fyne.NewMenuItem(c.tr.T(i18n.MenuPasteNow), c.startPasteFromClipboard),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem(c.tr.T(i18n.MenuCheckUpdate), func() { c.checkUpdate(c.window) }),
		fyne.NewMenuItem(c.tr.T(i18n.MenuDownloadPortable), func() { c.downloadLatest("portable") }),
		fyne.NewMenuItem(c.tr.T(i18n.MenuDownloadInstaller), func() { c.downloadLatest("installer") }),
		fyne.NewMenuItem(c.tr.T(i18n.MenuRepository), c.openRepository),
		fyne.NewMenuItemSeparator(),
		quit,
	)
	desk.SetSystemTrayMenu(menu)
	desk.SetSystemTrayWindow(c.window)
}

func (c *controller) registerHotkey() error {
	mods, key, err := parseHotkey(c.cfg.Hotkey)
	if err != nil {
		return err
	}
	requested := c.cfg.HotkeyString()

	c.hotkeyMu.Lock()
	defer c.hotkeyMu.Unlock()
	if c.hotkey != nil && c.registeredHotkey == requested {
		return nil
	}

	previous := c.registeredHotkey
	c.unregisterHotkeyLocked()

	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		if previous != "" {
			if restoreErr := c.registerHotkeyStringLocked(previous); restoreErr != nil {
				return fmt.Errorf("%w; restore previous hotkey %s: %v", err, previous, restoreErr)
			}
		}
		return err
	}
	c.setHotkeyLocked(hk, requested)
	return nil
}

func (c *controller) registerHotkeyStringLocked(value string) error {
	cfg := config.Default()
	if err := cfg.Set("hotkey", value); err != nil {
		return err
	}
	mods, key, err := parseHotkey(cfg.Hotkey)
	if err != nil {
		return err
	}
	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		return err
	}
	c.setHotkeyLocked(hk, cfg.HotkeyString())
	return nil
}

func (c *controller) setHotkeyLocked(hk *hotkey.Hotkey, value string) {
	done := make(chan struct{})
	c.hotkey = hk
	c.hotkeyDone = done
	c.registeredHotkey = value
	go c.listenHotkey(hk, done)
}

func (c *controller) unregisterHotkeyLocked() {
	if c.hotkeyDone != nil {
		close(c.hotkeyDone)
		c.hotkeyDone = nil
	}
	if c.hotkey != nil {
		_ = c.hotkey.Unregister()
		c.hotkey = nil
	}
	c.registeredHotkey = ""
}

func (c *controller) unregisterHotkey() {
	c.hotkeyMu.Lock()
	defer c.hotkeyMu.Unlock()
	c.unregisterHotkeyLocked()
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
		c.setStatus(c.tr.T(i18n.StatusClipboardEmpty))
		_ = c.driver.NotifyError()
		return
	}

	c.unregisterHotkey()
	c.setStatus(c.tr.T(i18n.StatusPasting))
	go func() {
		defer c.pasting.Store(false)
		defer func() {
			if err := c.registerHotkey(); err != nil {
				c.setStatus(c.tr.T(i18n.StatusPasteFinishedHotkeyFailed, err.Error()))
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
			c.setStatus(c.tr.T(i18n.StatusPasteFailed, err.Error()))
			return
		}
		c.setStatus(c.tr.T(i18n.StatusPasteFinished))
	}()
}

func (c *controller) checkUpdate(parent fyne.Window) {
	c.setStatus(c.tr.T(i18n.StatusCheckingUpdate))
	go func() {
		client := update.NewClient(c.cfg.Update.Repository)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		release, err := client.Latest(ctx)
		if err != nil {
			c.setStatus(c.tr.T(i18n.StatusUpdateCheckFailed, err.Error()))
			fyne.Do(func() { dialog.ShowError(err, parent) })
			return
		}
		if update.HasUpdate(metadata.Version, release) {
			msg := c.tr.T(i18n.StatusUpdateAvailable, metadata.Version, release.TagName)
			c.setStatus(msg)
			fyne.Do(func() { dialog.ShowInformation(c.tr.T(i18n.DialogUpdateTitle), msg, parent) })
			return
		}
		msg := c.tr.T(i18n.StatusCurrentVersionUpToDate, metadata.Version)
		c.setStatus(msg)
		fyne.Do(func() { dialog.ShowInformation(c.tr.T(i18n.DialogUpdateTitle), msg, parent) })
	}()
}

func (c *controller) downloadLatest(kind string) {
	c.setStatus(c.tr.T(i18n.StatusDownloading, kind))
	go func() {
		client := update.NewClient(c.cfg.Update.Repository)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		release, err := client.Latest(ctx)
		if err != nil {
			c.setStatus(c.tr.T(i18n.StatusDownloadFailed, err.Error()))
			return
		}
		asset, err := update.SelectAsset(release, kind, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			c.setStatus(c.tr.T(i18n.StatusDownloadFailed, err.Error()))
			return
		}
		path, err := client.Download(ctx, asset, "")
		if err != nil {
			c.setStatus(c.tr.T(i18n.StatusDownloadFailed, err.Error()))
			return
		}
		c.setStatus(c.tr.T(i18n.StatusDownloaded, path))
		c.app.SendNotification(&fyne.Notification{Title: metadata.Name, Content: c.tr.T(i18n.StatusDownloaded, path)})
	}()
}

func (c *controller) openRepository() {
	u, err := url.Parse(metadata.Repository)
	if err != nil {
		c.setStatus(c.tr.T(i18n.StatusInvalidRepositoryURL, err.Error()))
		return
	}
	if err := c.app.OpenURL(u); err != nil {
		c.setStatus(c.tr.T(i18n.StatusOpenRepositoryFailed, err.Error()))
	}
}

func (c *controller) refreshDiagnostics() {
	lines := []string{c.tr.T(i18n.DiagConfig) + ": " + c.cfgPath, c.tr.T(i18n.DiagDriver) + ": " + c.driver.Name()}
	for _, check := range c.driver.Check(context.Background()) {
		lines = append(lines, "["+string(check.Status)+"] "+check.Name+": "+check.Detail)
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
