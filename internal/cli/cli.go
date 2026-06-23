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
	"github.com/Mai-xiyu/Paste-Tool/internal/metadata"
	"github.com/Mai-xiyu/Paste-Tool/internal/platform"
	"github.com/Mai-xiyu/Paste-Tool/internal/update"
	"golang.design/x/clipboard"
)

func Run(args []string, stdout, stderr io.Writer) int {
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
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runPaste(args []string, stdout, stderr io.Writer) int {
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
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 1
	}
	input, err := readPasteInput(*source, *text, *fileName, fs.Args(), os.Stdin)
	if err != nil {
		fmt.Fprintf(stderr, "paste input: %v\n", err)
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
			fmt.Fprintf(stderr, "dry-run: %v\n", err)
			return 1
		}
		fmt.Fprint(stdout, recorder.String())
		return 0
	}

	driver := platform.NewDriver()
	if err := core.PasteText(ctx, input, options, driver); err != nil {
		fmt.Fprintf(stderr, "paste: %v\n", err)
		return 1
	}
	return 0
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, path, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%s %s\n", metadata.Name, metadata.Version)
	fmt.Fprintf(stdout, "config: %s\n", path)
	fmt.Fprintf(stdout, "hotkey: %s\n", cfg.HotkeyString())
	fmt.Fprintf(stdout, "paste: start_delay=%dms inter_key=%dms batch_size=%d batch_pause=%dms\n",
		cfg.Paste.StartDelayMS, cfg.Paste.InterKeyDelayMS, cfg.Paste.BatchSize, cfg.Paste.BatchPauseMS)

	driver := platform.NewDriver()
	fmt.Fprintf(stdout, "driver: %s\n", driver.Name())
	exitCode := 0
	for _, check := range driver.Check(context.Background()) {
		fmt.Fprintf(stdout, "[%s] %s: %s\n", check.Status, check.Name, check.Detail)
		if check.Status == platform.StatusError || check.Status == platform.StatusUnsupported {
			exitCode = 1
		}
	}
	if err := clipboard.Init(); err != nil {
		fmt.Fprintf(stdout, "[warning] clipboard: %v\n", err)
	} else {
		fmt.Fprintln(stdout, "[ok] clipboard: text clipboard backend initialized")
	}
	return exitCode
}

func runConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "config requires get, set, path, or reset")
		return 2
	}
	cfg, path, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
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
			fmt.Fprintf(stderr, "config get: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, value)
		return 0
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(stderr, "usage: paste-tool config set <key> <value>")
			return 2
		}
		if err := cfg.Set(args[1], strings.Join(args[2:], " ")); err != nil {
			fmt.Fprintf(stderr, "config set: %v\n", err)
			return 1
		}
		if err := config.Save(path, cfg); err != nil {
			fmt.Fprintf(stderr, "config save: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "saved %s\n", path)
		return 0
	case "reset":
		if err := config.Save(path, config.Default()); err != nil {
			fmt.Fprintf(stderr, "config reset: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "reset %s\n", path)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown config command %q\n", args[0])
		return 2
	}
}

func runUpdate(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "update requires check or download")
		return 2
	}
	cfg, _, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 1
	}
	client := update.NewClient(cfg.Update.Repository)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	switch args[0] {
	case "check":
		release, err := client.Latest(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "update check: %v\n", err)
			return 1
		}
		if update.HasUpdate(metadata.Version, release) {
			fmt.Fprintf(stdout, "update available: %s -> %s\n%s\n", metadata.Version, release.TagName, release.HTMLURL)
		} else {
			fmt.Fprintf(stdout, "current version %s is up to date against latest %s\n", metadata.Version, release.TagName)
		}
		return 0
	case "download":
		return runUpdateDownload(args[1:], client, ctx, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown update command %q\n", args[0])
		return 2
	}
}

func runUpdateDownload(args []string, client update.Client, ctx context.Context, stdout, stderr io.Writer) int {
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
		fmt.Fprintf(stderr, "update download: %v\n", err)
		return 1
	}
	asset, err := update.SelectAsset(release, *kind, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fmt.Fprintf(stderr, "select asset: %v\n", err)
		return 1
	}
	path, err := client.Download(ctx, asset, *outputDir)
	if err != nil {
		fmt.Fprintf(stderr, "download asset: %v\n", err)
		return 1
	}
	abs, _ := filepath.Abs(path)
	fmt.Fprintf(stdout, "downloaded %s\n", abs)
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

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  paste-tool                         Launch tray GUI
  paste-tool gui                     Launch tray GUI
  paste-tool paste [flags]           Type text into the focused target
  paste-tool doctor                  Print platform and config diagnostics
  paste-tool config get [key]        Print config
  paste-tool config set <key> <val>  Update config
  paste-tool update check            Check GitHub latest release
  paste-tool update download [kind]  Download latest portable or installer
  paste-tool version                 Print version`)
}
