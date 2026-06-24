package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Mai-xiyu/Paste-Tool/internal/config"
	"github.com/Mai-xiyu/Paste-Tool/internal/core"
	"github.com/Mai-xiyu/Paste-Tool/internal/gui"
	"github.com/Mai-xiyu/Paste-Tool/internal/i18n"
	"github.com/Mai-xiyu/Paste-Tool/internal/metadata"
	"github.com/Mai-xiyu/Paste-Tool/internal/platform"
	"github.com/Mai-xiyu/Paste-Tool/internal/update"
	"golang.design/x/clipboard"
)

func Run(args []string, stdout, stderr io.Writer) int {
	tr := translatorFromDefaultConfig()
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if len(args) == 0 {
		if err := gui.Run(context.Background()); err != nil {
			fmt.Fprintf(stderr, "gui: %v\n", err)
			return 1
		}
		return 0
	}

	switch args[0] {
	case "gui":
		if err := gui.Run(context.Background()); err != nil {
			fmt.Fprintf(stderr, "gui: %v\n", err)
			return 1
		}
		return 0
	case "paste":
		return runPaste(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "config":
		return runConfig(args[1:], stdout, stderr)
	case "update":
		return runUpdate(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "%s %s %s/%s\n", metadata.Name, metadata.Version, runtime.GOOS, runtime.GOARCH)
		return 0
	case "help", "-h", "--help":
		printUsage(stdout, tr)
		return 0
	default:
		fmt.Fprintf(stderr, tr.T(i18n.CLIUnknownCommand)+"\n\n", args[0])
		printUsage(stderr, tr)
		return 2
	}
}

func runPaste(args []string, stdout, stderr io.Writer) int {
	tr := translatorFromDefaultConfig()
	fs := flag.NewFlagSet("paste", flag.ContinueOnError)
	fs.SetOutput(stderr)
	source := fs.String("source", "clipboard", "clipboard, stdin, file, or arg")
	text := fs.String("text", "", "text used when --source=arg")
	fileName := fs.String("file", "", "file used when --source=file")
	dryRun := fs.Bool("dry-run", false, "print normalized text instead of injecting keyboard input")
	timeout := fs.Duration("timeout", 5*time.Minute, "maximum paste duration")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, _, err := config.LoadDefault()
	tr = i18n.New(cfg.UI.Language)
	if err != nil {
		fmt.Fprintf(stderr, tr.T(i18n.CLIConfigLoadFailed)+"\n", err)
		return 1
	}
	input, err := readPasteInput(*source, *text, *fileName, fs.Args(), os.Stdin)
	if err != nil {
		fmt.Fprintf(stderr, tr.T(i18n.CLIPasteInputFailed)+"\n", err)
		return 1
	}
	options := core.Options{
		StartDelay:    cfg.Paste.StartDelay(),
		InterKeyDelay: cfg.Paste.InterKeyDelay(),
		BatchSize:     cfg.Paste.BatchSize,
		BatchPause:    cfg.Paste.BatchPause(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if *dryRun {
		recorder := &core.Recorder{}
		if err := core.PasteTextWithSleeper(ctx, input, options, recorder, noSleep{}); err != nil {
			fmt.Fprintf(stderr, tr.T(i18n.CLIDryRunFailed)+"\n", err)
			return 1
		}
		fmt.Fprint(stdout, recorder.String())
		return 0
	}

	driver := platform.NewDriver()
	if err := core.PasteText(ctx, input, options, driver); err != nil {
		fmt.Fprintf(stderr, tr.T(i18n.CLIPasteFailed)+"\n", err)
		return 1
	}
	return 0
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	tr := translatorFromDefaultConfig()
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, path, err := config.LoadDefault()
	tr = i18n.New(cfg.UI.Language)
	if err != nil {
		fmt.Fprintf(stderr, tr.T(i18n.CLIConfigLoadFailed)+"\n", err)
		return 1
	}
	fmt.Fprintln(stdout, tr.T(i18n.CLIDoctorHeader, metadata.Name, metadata.Version))
	fmt.Fprintln(stdout, tr.T(i18n.CLIDoctorConfig, path))
	fmt.Fprintln(stdout, tr.T(i18n.CLIDoctorHotkey, cfg.HotkeyString()))
	fmt.Fprintln(stdout, tr.T(
		i18n.CLIDoctorPaste,
		cfg.Paste.StartDelayMS,
		cfg.Paste.InterKeyDelayMS,
		cfg.Paste.BatchSize,
		cfg.Paste.BatchPauseMS,
	))

	driver := platform.NewDriver()
	fmt.Fprintln(stdout, tr.T(i18n.CLIDoctorDriver, driver.Name()))
	exitCode := 0
	for _, check := range driver.Check(context.Background()) {
		fmt.Fprintf(stdout, "[%s] %s: %s\n", check.Status, check.Name, check.Detail)
		if check.Status == platform.StatusError || check.Status == platform.StatusUnsupported {
			exitCode = 1
		}
	}
	if err := clipboard.Init(); err != nil {
		fmt.Fprintln(stdout, tr.T(i18n.CLIDoctorClipboardWarning, err))
	} else {
		fmt.Fprintln(stdout, tr.T(i18n.CLIDoctorClipboardOK))
	}
	return exitCode
}

func runConfig(args []string, stdout, stderr io.Writer) int {
	tr := translatorFromDefaultConfig()
	if len(args) == 0 {
		fmt.Fprintln(stderr, tr.T(i18n.CLIConfigRequiresCommand))
		return 2
	}
	cfg, path, err := config.LoadDefault()
	tr = i18n.New(cfg.UI.Language)
	if err != nil {
		fmt.Fprintf(stderr, tr.T(i18n.CLIConfigLoadFailed)+"\n", err)
		return 1
	}
	switch args[0] {
	case "path":
		fmt.Fprintln(stdout, path)
		return 0
	case "get":
		key := ""
		if len(args) > 1 {
			key = args[1]
		}
		value, err := cfg.Get(key)
		if err != nil {
			fmt.Fprintf(stderr, tr.T(i18n.CLIConfigGetFailed)+"\n", err)
			return 1
		}
		fmt.Fprintln(stdout, value)
		return 0
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(stderr, tr.T(i18n.CLIConfigSetUsage))
			return 2
		}
		if err := cfg.Set(args[1], strings.Join(args[2:], " ")); err != nil {
			fmt.Fprintf(stderr, tr.T(i18n.CLIConfigSetFailed)+"\n", err)
			return 1
		}
		if err := config.Save(path, cfg); err != nil {
			fmt.Fprintf(stderr, tr.T(i18n.CLIConfigSaveFailed)+"\n", err)
			return 1
		}
		fmt.Fprintln(stdout, tr.T(i18n.CLIConfigSaved, path))
		return 0
	case "reset":
		if err := config.Save(path, config.Default()); err != nil {
			fmt.Fprintf(stderr, tr.T(i18n.CLIConfigResetFailed)+"\n", err)
			return 1
		}
		fmt.Fprintln(stdout, tr.T(i18n.CLIConfigReset, path))
		return 0
	default:
		fmt.Fprintf(stderr, tr.T(i18n.CLIUnknownConfigCommand)+"\n", args[0])
		return 2
	}
}

func runUpdate(args []string, stdout, stderr io.Writer) int {
	tr := translatorFromDefaultConfig()
	if len(args) == 0 {
		fmt.Fprintln(stderr, tr.T(i18n.CLIUpdateRequiresCommand))
		return 2
	}
	cfg, _, err := config.LoadDefault()
	tr = i18n.New(cfg.UI.Language)
	if err != nil {
		fmt.Fprintf(stderr, tr.T(i18n.CLIConfigLoadFailed)+"\n", err)
		return 1
	}
	client := update.NewClient(cfg.Update.Repository)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	switch args[0] {
	case "check":
		release, err := client.Latest(ctx)
		if err != nil {
			fmt.Fprintf(stderr, tr.T(i18n.CLIUpdateCheckFailed)+"\n", err)
			return 1
		}
		if update.HasUpdate(metadata.Version, release) {
			fmt.Fprintln(stdout, tr.T(i18n.CLIUpdateAvailable, metadata.Version, release.TagName, release.HTMLURL))
		} else {
			fmt.Fprintln(stdout, tr.T(i18n.CLIUpdateUpToDate, metadata.Version, release.TagName))
		}
		return 0
	case "download":
		return runUpdateDownload(args[1:], client, ctx, stdout, stderr, tr)
	default:
		fmt.Fprintf(stderr, tr.T(i18n.CLIUnknownUpdateCommand)+"\n", args[0])
		return 2
	}
}

func runUpdateDownload(args []string, client update.Client, ctx context.Context, stdout, stderr io.Writer, tr i18n.Translator) int {
	fs := flag.NewFlagSet("update download", flag.ContinueOnError)
	fs.SetOutput(stderr)
	kind := fs.String("kind", "portable", "portable or installer")
	outputDir := fs.String("output", "", "output directory; defaults to Downloads")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		*kind = fs.Arg(0)
	}
	release, err := client.Latest(ctx)
	if err != nil {
		fmt.Fprintf(stderr, tr.T(i18n.CLIUpdateCheckFailed)+"\n", err)
		return 1
	}
	asset, err := update.SelectAsset(release, *kind, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fmt.Fprintf(stderr, tr.T(i18n.CLIUpdateSelectAssetFailed)+"\n", err)
		return 1
	}
	path, err := client.Download(ctx, asset, *outputDir)
	if err != nil {
		fmt.Fprintf(stderr, tr.T(i18n.CLIUpdateDownloadAssetFailed)+"\n", err)
		return 1
	}
	abs, _ := filepath.Abs(path)
	fmt.Fprintln(stdout, tr.T(i18n.CLIUpdateDownloaded, abs))
	return 0
}

func readPasteInput(source, text, fileName string, extra []string, stdin io.Reader) (string, error) {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "clipboard", "":
		if err := clipboard.Init(); err != nil {
			return "", fmt.Errorf("initialize clipboard: %w", err)
		}
		return string(clipboard.Read(clipboard.FmtText)), nil
	case "stdin":
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "file":
		if strings.TrimSpace(fileName) == "" {
			return "", errors.New("--file is required when --source=file")
		}
		data, err := os.ReadFile(fileName)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "arg":
		if text != "" {
			return text, nil
		}
		if len(extra) > 0 {
			return strings.Join(extra, " "), nil
		}
		return "", errors.New("--text or trailing arguments are required when --source=arg")
	default:
		return "", fmt.Errorf("unknown source %q", source)
	}
}

type noSleep struct{}

func (noSleep) Sleep(ctx context.Context, _ time.Duration) error {
	return ctx.Err()
}

func printUsage(w io.Writer, tr i18n.Translator) {
	fmt.Fprintln(w, tr.T(i18n.CLIUsage))
}

func translatorFromDefaultConfig() i18n.Translator {
	cfg, _, err := config.LoadDefault()
	if err != nil {
		return i18n.New("auto")
	}
	return i18n.New(cfg.UI.Language)
}
