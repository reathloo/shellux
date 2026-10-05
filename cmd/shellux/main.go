package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/reathloo/shellux/internal/animation"
	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/live"
	"github.com/reathloo/shellux/internal/locale"
	"github.com/reathloo/shellux/internal/render"
	"github.com/reathloo/shellux/internal/settingsui"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/terminal"
	"github.com/reathloo/shellux/internal/theme"
	"github.com/reathloo/shellux/internal/themepack"
	"github.com/reathloo/shellux/internal/version"
)

func main() {
	if runCommand(os.Args[1:]) {
		return
	}

	showVersion := flag.Bool("version", false, "print the version")
	watch := flag.Bool("watch", false, "run a foreground live preview until interrupted")
	once := flag.Bool("once", false, "render the header once and exit")
	inPlace := flag.Bool("in-place", false, "update a reserved header without moving the shell cursor")
	reserveHeader := flag.Bool("reserve-header", false, "reserve fixed header rows for shell integration")
	clearScrollback := flag.Bool("clear-scrollback", false, "clear existing terminal scrollback before reserving the initial header")
	preserveContent := flag.Bool("preserve-content", false, "redraw reserved header without clearing terminal output")
	parentPID := flag.Int("parent-pid", 0, "stop watch mode when this parent process exits")
	interval := flag.Duration("interval", time.Second, "refresh interval in watch mode")
	animationName := flag.String("animation", config.Default().Animation, "animation to use ("+strings.Join(config.Animations(), ", ")+")")
	styleName := flag.String("style", config.Default().Style, "visual style to use ("+strings.Join(config.Styles(), ", ")+")")
	configPath := flag.String("config", "", "path to a JSON configuration file")
	flag.Parse()
	if *watch && *once {
		fmt.Fprintln(os.Stderr, "shellux: --watch and --once cannot be used together")
		os.Exit(2)
	}
	if *inPlace && !*watch {
		fmt.Fprintln(os.Stderr, "shellux: --in-place requires --watch")
		os.Exit(2)
	}
	if *reserveHeader && *watch {
		fmt.Fprintln(os.Stderr, "shellux: --reserve-header cannot be used with --watch")
		os.Exit(2)
	}
	if *clearScrollback && !*reserveHeader {
		fmt.Fprintln(os.Stderr, "shellux: --clear-scrollback requires --reserve-header")
		os.Exit(2)
	}
	if *preserveContent && !*reserveHeader {
		fmt.Fprintln(os.Stderr, "shellux: --preserve-content requires --reserve-header")
		os.Exit(2)
	}
	if *preserveContent && *clearScrollback {
		fmt.Fprintln(os.Stderr, "shellux: --preserve-content cannot be used with --clear-scrollback")
		os.Exit(2)
	}

	if *showVersion {
		fmt.Println(version.String())
		return
	}
	settings, err := loadSettings(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shellux: load config: %v\n", err)
		os.Exit(2)
	}
	animationSet := false
	styleSet := false
	intervalSet := false
	flag.Visit(func(item *flag.Flag) {
		animationSet = animationSet || item.Name == "animation"
		styleSet = styleSet || item.Name == "style"
		intervalSet = intervalSet || item.Name == "interval"
	})
	if !animationSet {
		*animationName = settings.Animation
	}
	if !styleSet {
		*styleName = settings.Style
	}
	if !intervalSet {
		*interval = settings.Interval()
	}
	if !isAnimation(*animationName) {
		fmt.Fprintf(os.Stderr, "shellux: unknown animation %q\n", *animationName)
		os.Exit(2)
	}
	var selectedAnimation animation.Animation = animation.ShelluxDefault{}
	if *animationName != "shelluxdefault" {
		if p, err := themepack.LoadName(*animationName); err == nil {
			selectedAnimation = p
		} else {
			*styleName = "shelluxdefault"
			settings.Background = "#000000"
		}
	}
	if *styleName != *animationName {
		if _, ok := themepack.Lookup(*styleName); ok {
			if _, err := themepack.LoadName(*styleName); err != nil {
				*styleName = "shelluxdefault"
			}
		}
	}
	selectedStyle, err := theme.ByName(*styleName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shellux: %v\n", err)
		os.Exit(2)
	}

	emitBackground(settings.Background)
	if *watch {
		watchHeader(*interval, settings.Visible, *inPlace, *parentPID, selectedAnimation, selectedStyle, locale.Resolve(settings.Language))
		return
	}

	snapshot, err := systeminfo.SnapshotInfoWithDisplay(settings.Visible)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shellux: collect system information: %v\n", err)
		os.Exit(1)
	}
	layout := render.NewHeaderLayout(snapshot, animation.Frame(selectedAnimation, 0), selectedStyle, settings.Visible, locale.Resolve(settings.Language))
	if *reserveHeader {
		if rows, columns, sizeErr := terminal.Size(os.Stdout.Fd()); sizeErr == nil {
			layout = layout.FitTerminal(columns, rows)
		}
		regions := []string{terminal.Region(layout.EyeRegion(), 1, 1, layout.EyeWidth)}
		if len(layout.Dashboard) > 0 {
			regions = append(regions, terminal.Region(layout.Dashboard, 1, layout.DashboardColumn, layout.DashboardWidth))
		}
		output := terminal.PinnedRegions(layout.Height(), regions...)
		if *preserveContent {
			output = terminal.PinnedRegionsPreservingContent(layout.Height(), regions...)
		}
		if *clearScrollback {
			output += terminal.EraseScrollback()
		}
		fmt.Print(output)
		return
	}
	fmt.Print(layout.String())
}

func runCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if strings.HasPrefix(args[0], "-") {
		return false
	}
	switch args[0] {
	case "settings":
		if len(args) != 1 {
			commandUsage("shellux settings (no additional arguments)")
		}
		code, settings, err := settingsui.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "shellux: settings: %v\n", err)
		}
		if code == settingsui.Saved {
			emitBackground(settings.Background)
		}
		if code != 0 {
			os.Exit(code)
		}
		return true
	case "background":
		if len(args) == 1 {
			settings, err := config.LoadDefault()
			if err != nil {
				commandError("read background", err)
			}
			color := settings.Background
			if color == "" {
				color = "default"
			}
			fmt.Printf("Background: %s\nColors: %s; or '#RRGGBB'\nUse 'shellux background default' to restore the terminal profile color.\n", config.DisplayBackground(color), strings.Join(config.BackgroundNames(), ", "))
			return true
		}
		if len(args) != 2 {
			commandUsage("shellux background <color>|default")
		}
		selected, path, err := saveAppearance("background", args[1])
		color := selected.Background
		if err != nil {
			commandError("set background", err)
		}
		applied := emitBackground(color)
		fmt.Printf("shellux: background %s saved to %s\n", color, path)
		if applied {
			fmt.Println("shellux: color request sent to terminal; unsupported terminals ignore it")
		}
		return true
	case "show":
		if len(args) == 1 {
			printMenu()
			return true
		}
		if args[1] == "set" || args[1] == "replace" {
			path, err := config.UpdateMenu(args[1], args[2:])
			if err != nil {
				commandError("change selection", err)
			}
			fmt.Printf("shellux: display settings saved to %s\n", path)
			return true
		}
		if len(args) != 3 && len(args) != 4 {
			commandUsage("shellux show <item> [bar|percent|value] on|off")
		}
		state := args[len(args)-1]
		if state != "on" && state != "off" {
			commandUsage("shellux show <item> [bar|percent|value] on|off")
		}
		part := ""
		if len(args) == 4 {
			part = args[2]
		}
		path, err := config.SetDisplay(args[1], part, state == "on")
		if err != nil {
			commandError("change display", err)
		}
		fmt.Printf("shellux: display settings saved to %s\n", path)
		return true
	case "menu":
		if len(args) == 1 || (len(args) == 2 && args[1] == "list") {
			printMenu()
			return true
		}
		if len(args) != 2 || args[1] != "reset" {
			commandUsage("shellux menu list|reset; use 'shellux show <item> on|off', 'shellux show replace <old> <new>', or 'shellux show set [items...]' to change entries")
		}
		path, err := config.UpdateMenu("reset", nil)
		if err != nil {
			commandError("reset menu", err)
		}
		fmt.Printf("shellux: menu saved to %s\n", path)
		return true
	case "theme":
		switch {
		case len(args) == 2 && args[1] == "new":
			commandUsage("shellux theme new <name>")
		case len(args) == 2 && args[1] == "delete":
			commandUsage("shellux theme delete <name>")
		case len(args) == 3 && args[1] == "new":
			name, path, err := config.SaveCurrentTheme(args[2])
			if err != nil {
				commandError("save theme", err)
			}
			fmt.Printf("shellux: custom theme %q saved to %s\n", name, path)
			return true
		case len(args) == 3 && args[1] == "delete":
			name, path, err := config.DeleteTheme(args[2])
			if err != nil {
				commandError("delete theme", err)
			}
			fmt.Printf("shellux: custom theme %q deleted from %s\n", name, path)
			return true
		case len(args) == 2:
			selected, path, err := saveAppearance("theme", args[1])
			if err != nil {
				commandError("set theme", err)
			}
			emitBackground(selected.Background)
			fmt.Printf("shellux: theme %q saved (animation: %s, style: %s, background: %s) to %s\n", args[1], selected.Animation, config.DisplayStyle(selected.Style), selected.Background, path)
			return true
		default:
			commandUsage("shellux theme <name>|default|random|new <name>|delete <name>")
		}
	case "themes":
		if len(args) != 1 {
			commandUsage("shellux themes")
		}
		themes, err := config.Themes()
		if err != nil {
			commandError("list themes", err)
		}
		fmt.Println("Available themes:")
		for _, saved := range themes {
			kind := "built-in"
			if _, ok := themepack.Lookup(saved.Name); ok {
				kind = "download"
				if themepack.IsInstalled(saved.Name) {
					kind = "installed"
				}
			}
			if saved.Custom {
				kind = "custom"
			}
			if saved.Name == "default" {
				kind = "default"
			}
			fmt.Printf("  %-12s %-9s animation: %-8s style: %-8s background: %s\n", saved.Name, "("+kind+")", saved.Animation, saved.Style, saved.Background)
		}
		return true
	case "style":
		if len(args) != 2 {
			commandUsage("shellux style <name>")
		}
		_, path, err := saveAppearance("style", args[1])
		if err != nil {
			commandError("set style", err)
		}
		fmt.Printf("shellux: style %q saved to %s\n", args[1], path)
		return true
	case "styles":
		if len(args) != 1 {
			commandUsage("shellux styles")
		}
		fmt.Printf("Available styles:\n  %s\n", strings.Join(config.Styles(), "\n  "))
		return true
	case "animation":
		if len(args) != 2 {
			commandUsage("shellux animation <name>")
		}
		_, path, err := saveAppearance("animation", args[1])
		if err != nil {
			commandError("set animation", err)
		}
		fmt.Printf("shellux: animation %q saved to %s\n", args[1], path)
		return true
	case "animations":
		if len(args) != 1 {
			commandUsage("shellux animations")
		}
		fmt.Printf("Available animations:\n  %s\n", strings.Join(config.Animations(), "\n  "))
		return true
	case "help":
		if len(args) != 1 {
			commandUsage("shellux help")
		}
		fmt.Print(commandHelp)
		return true
	case "status":
		if len(args) != 1 {
			commandUsage("shellux status")
		}
		settings, err := config.LoadDefault()
		if err != nil {
			commandError("read status", err)
		}
		themes, err := config.Themes()
		if err != nil {
			commandError("read status", err)
		}
		fmt.Print(statusReport(settings, themes))
		return true
	case "doctor":
		if len(args) != 1 {
			commandUsage("shellux doctor")
		}
		if !doctor() {
			os.Exit(1)
		}
		return true
	case "off", "on":
		if len(args) != 1 {
			commandUsage("shellux " + args[0])
		}
		commandError(args[0], fmt.Errorf("requires the sourced Zsh or Bash integration"))
	case "reload":
		commandError("reload", fmt.Errorf("requires the sourced Zsh or Bash integration"))
	case "exit":
		commandError("exit", fmt.Errorf("requires the sourced Zsh or Bash integration"))
	case "thermal":
		if len(args) != 2 || args[1] != "enable" {
			commandUsage("shellux thermal enable")
		}
		completed, err := systeminfo.EnableThermalAccess()
		if err != nil {
			commandError("enable thermal access", err)
		}
		if completed {
			if _, err := config.SetDisplay("thermal", "", true); err != nil {
				commandError("show thermal state", err)
			}
			fmt.Println("shellux: thermal access enabled and indicator shown")
		}
		return true
	}
	fmt.Fprintf(os.Stderr, "shellux: command not found: %q; run 'shellux help' for all commands\n", args[0])
	os.Exit(2)
	return true
}

func emitBackground(color string) bool {
	if color == "" {
		return false
	}
	normalized, err := config.BackgroundColor(color)
	if err != nil {
		return false
	}
	if _, _, err := terminal.Size(os.Stdout.Fd()); err != nil {
		return false
	}
	fmt.Print(terminal.Background(normalized))
	return true
}

func printMenu() {
	settings, err := config.LoadDefault()
	if err != nil {
		commandError("read menu", err)
	}
	fmt.Print(menuReport(settings))
}

func menuReport(settings config.Config) string {
	var output strings.Builder
	fmt.Fprintln(&output, "Menu items (maximum 11 slots; battery + thermal and spotify + progress share slots):")
	state := func(key string) string {
		if config.Shown(settings.Visible, key) {
			return "on"
		}
		return "off"
	}
	for _, name := range config.MenuItems() {
		fmt.Fprintf(&output, "  %-16s %s", config.DisplayMenuItem(name), state(name))
		for _, part := range config.DisplayParts(name) {
			fmt.Fprintf(&output, "  %s: %s", part, state(name+"."+part))
		}
		fmt.Fprintln(&output)
	}
	return output.String()
}

const commandHelp = `Shellux commands:
  shellux settings               visual settings menu (interactive terminal)
                                  Tab: focus, arrows: select, Space: toggle
                                  Ctrl+S: save and reload header (clear scrollback)
                                  Esc: cancel, Ctrl+C: discard
  shellux                         redraw the fixed header
  shellux reload                  restart the live header
  shellux off                     hide the header in this terminal tab
  shellux on                      restore the header in this terminal tab
  shellux status                  show current theme and refresh settings
  shellux doctor                  check setup and renderer state
  shellux exit                    exit the current shell

  shellux theme <name>            set matching animation, style and background
  shellux theme default           restore the default theme
  shellux theme random            random animation/style with matching background
  shellux theme new <name>        save animation, style and background as a theme
  shellux theme delete <name>     delete a custom theme
  shellux themes                  list built-in and custom themes

  shellux style <name>            change only the menu style
  shellux styles                  list styles
  shellux animation <name>        change only the animation
  shellux animations              list animations

  shellux background <color>      set named or '#RRGGBB' background (OSC 11 terminals)
  shellux background default      restore the terminal profile background
  shellux background              show saved background and available colors

  shellux show                    list entry and component visibility
  shellux show <item> on|off       show or hide a whole entry
  shellux show <item> <part> on|off toggle a component (see shellux show)
  shellux show date timezone on    show the local timezone name
  shellux show date utc on         show the current UTC offset
  shellux show battery remaining on show estimated battery runtime
  shellux show cpu top on          show the process with the highest CPU average
  shellux show ram pressure on     show memory pressure
  shellux show status connection on show the connection type
  shellux show status name on      show the network name, when available
  shellux show traffic upload on   show upload rate within traffic
  shellux show traffic download off hide download rate within traffic
  shellux show traffic latency on  show gateway ping within traffic

  shellux show replace <old> <new> exchange two entries
  shellux show set [items...]      select only these entries (empty hides all)

  shellux menu list               list available and selected menu items
  shellux menu reset              restore the default selection
  shellux thermal enable          authorize and show the macOS thermal state
`

func statusReport(settings config.Config, themes []config.ThemeInfo) string {
	matching := make([]string, 0)
	for _, candidate := range themes {
		if candidate.Animation == settings.Animation && candidate.Style == config.DisplayStyle(settings.Style) && candidate.Background == settings.Background {
			matching = append(matching, candidate.Name)
		}
	}
	sort.Strings(matching)
	themeName := "custom combination"
	if len(matching) > 0 {
		themeName = strings.Join(matching, ", ")
	}
	return fmt.Sprintf("Theme match:     %s\nAnimation:       %s\nStyle:           %s\nBackground:      %s\nMetrics refresh: %s\nAnimation frame: %s\n",
		themeName, settings.Animation, config.DisplayStyle(settings.Style), config.DisplayBackground(settings.Background), settings.RefreshInterval, animationByName(settings.Animation).Interval())
}

func doctor() bool {
	return doctorReport(os.Stdout)
}

func doctorReport(out io.Writer) bool {
	ok := true
	if executable, err := os.Executable(); err == nil {
		fmt.Fprintf(out, "Binary:      OK (%s)\n", executable)
	} else {
		fmt.Fprintf(out, "Binary:      ERROR (%v)\n", err)
		ok = false
	}
	path, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintf(out, "Config:      ERROR (%v)\n", err)
		ok = false
	} else if _, err := config.LoadDefault(); err != nil {
		fmt.Fprintf(out, "Config:      ERROR (%s: %v)\n", path, err)
		ok = false
	} else if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(out, "Config:      OK (built-in defaults; no config file)")
	} else {
		fmt.Fprintf(out, "Config:      OK (%s)\n", path)
	}
	shell := os.Getenv("SHELLUX_DIAG_SHELL")
	if shell != "zsh" && shell != "bash" {
		fmt.Fprintln(out, "Integration: not detected; source shell/shellux.zsh or shell/shellux.bash")
		return false
	}
	fmt.Fprintf(out, "Integration: OK (%s)\n", shell)
	if os.Getenv("SHELLUX_DIAG_ENABLED") == "0" {
		fmt.Fprintln(out, "Header:      off in this terminal tab")
	} else if os.Getenv("SHELLUX_DIAG_WATCH") == "1" {
		fmt.Fprintln(out, "Renderer:    OK (running before diagnostics; paused for output)")
	} else if os.Getenv("SHELLUX_DIAG_WATCH") == "paused" {
		fmt.Fprintln(out, "Renderer:    paused for normal command output")
	} else {
		fmt.Fprintln(out, "Renderer:    not running before diagnostics")
		ok = false
	}
	return ok
}

func commandUsage(usage string) {
	fmt.Fprintf(os.Stderr, "usage: %s\nrun 'shellux help' for all commands\n", usage)
	os.Exit(2)
}

func commandError(action string, err error) {
	fmt.Fprintf(os.Stderr, "shellux: %s: %v\n", action, err)
	fmt.Fprintln(os.Stderr, "run 'shellux help' for all commands")
	os.Exit(2)
}

func loadSettings(path string) (config.Config, error) {
	if path != "" {
		return config.Load(path)
	}
	return config.LoadDefault()
}

func animationByName(name string) animation.Animation {
	if name != "shelluxdefault" {
		if p, err := themepack.LoadName(name); err == nil {
			return p
		}
	}
	return animation.ShelluxDefault{}
}

func isAnimation(name string) bool {
	for _, animationName := range config.Animations() {
		if name == animationName {
			return true
		}
	}
	return false
}

func watchHeader(interval time.Duration, visible map[string]bool, inPlace bool, parentPID int, selectedAnimation animation.Animation, selectedStyle theme.Theme, language locale.Language) {
	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancel(signalContext)
	defer cancel()
	reservedHeight := 0
	if inPlace {
		defer func() {
			fmt.Fprint(os.Stdout, terminal.UnpinHeader(reservedHeight))
		}()
	}
	resizeEvents := make(chan os.Signal, 1)
	if inPlace {
		signal.Notify(resizeEvents, syscall.SIGWINCH)
		defer signal.Stop(resizeEvents)
	}
	if parentPID > 0 {
		go cancelWhenProcessExits(ctx, cancel, parentPID)
	}

	frame := 0
	var snapshot systeminfo.Snapshot
	var snapshotMu sync.RWMutex
	columns := 0
	rows := 0
	refreshSize := func() bool {
		if updatedRows, updatedColumns, err := terminal.Size(os.Stdout.Fd()); err == nil {
			changed := (columns > 0 && updatedColumns != columns) || (rows > 0 && updatedRows != rows)
			rows = updatedRows
			columns = updatedColumns
			return changed
		}
		return false
	}
	refreshSize()
	layoutForFrame := func() render.HeaderLayout {
		snapshotMu.RLock()
		currentSnapshot := snapshot
		snapshotMu.RUnlock()
		layout := render.NewHeaderLayout(currentSnapshot, animation.Frame(selectedAnimation, frame), selectedStyle, visible, language)
		if inPlace {
			layout = layout.FitTerminal(columns, rows)
		}
		return layout
	}
	renderCompleteHeader := func(layout render.HeaderLayout) string {
		clearHeight := max(reservedHeight, layout.Height())
		output := terminal.MaintainHeader(layout.Height()) + terminal.ClearRows(clearHeight)
		output += terminal.Region(layout.EyeRegion(), 1, 1, layout.EyeWidth)
		output += headerGutter(layout)
		if len(layout.Dashboard) > 0 {
			output += terminal.Region(layout.Dashboard, 1, layout.DashboardColumn, layout.DashboardWidth)
		}
		reservedHeight = layout.Height()
		return output
	}
	runner := live.Runner{
		AnimationInterval: selectedAnimation.Interval(),
		MetricsInterval:   interval,
		Output:            os.Stdout,
		UpdateMetrics: func() error {
			updated, err := systeminfo.SnapshotInfoWithDisplay(visible)
			if err != nil {
				return err
			}
			snapshotMu.Lock()
			snapshot = updated
			snapshotMu.Unlock()
			return nil
		},
		AdvanceAnimation: func() {
			frame++
		},
		RenderInitial: func() (string, error) {
			layout := layoutForFrame()
			reservedHeight = layout.Height()
			if inPlace {
				return terminal.MaintainHeader(layout.Height()) +
					terminal.Region(layout.EyeRegion(), 1, 1, layout.EyeWidth) +
					headerGutter(layout) +
					terminal.Region(layout.Dashboard, 1, layout.DashboardColumn, layout.DashboardWidth), nil
			}
			return terminal.FullScreen(layout.String()), nil
		},
		RenderAnimation: func() (string, error) {
			resized := inPlace && refreshSize()
			layout := layoutForFrame()
			if resized {
				return renderCompleteHeader(layout), nil
			}
			prefix := ""
			if inPlace {
				prefix = terminal.MaintainHeader(layout.Height())
			}
			return prefix + terminal.Region(layout.EyeRegion(), 1, 1, layout.EyeWidth) + headerGutter(layout), nil
		},
		RenderMetrics: func() (string, error) {
			resized := inPlace && refreshSize()
			layout := layoutForFrame()
			if resized {
				return renderCompleteHeader(layout), nil
			}
			prefix := ""
			if inPlace {
				prefix = terminal.MaintainHeader(layout.Height())
			}
			return prefix + headerGutter(layout) + terminal.Region(layout.Dashboard, 1, layout.DashboardColumn, layout.DashboardWidth), nil
		},
		ResizeEvents: resizeEvents,
		RenderResize: func() (string, error) {
			refreshSize()
			layout := layoutForFrame()
			return renderCompleteHeader(layout), nil
		},
	}
	if err := runner.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "shellux: refresh header: %v\n", err)
	}
}

func headerGutter(layout render.HeaderLayout) string {
	if len(layout.Dashboard) == 0 {
		return ""
	}
	width := layout.DashboardColumn - layout.EyeWidth - 1
	if width < 1 {
		return ""
	}
	return terminal.Region([]string{""}, 1, layout.EyeWidth+1, width)
}

func cancelWhenProcessExits(ctx context.Context, cancel context.CancelFunc, processID int) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := syscall.Kill(processID, 0); err != nil {
				cancel()
				return
			}
		}
	}
}

// Downloads finish before the existing editor performs its conflict check/write.
func saveAppearance(field, value string) (config.Config, string, error) {
	session, err := config.OpenEditor()
	if err != nil {
		return config.Config{}, "", err
	}
	draft := session.Config.Clone()
	if field == "theme" {
		_, err = draft.ApplyTheme(value)
	} else {
		err = draft.ChangeAppearance(field, value)
	}
	if err != nil {
		return draft, "", err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := draft.EnsurePackages(ctx); err != nil {
		return draft, "", err
	}
	if err := session.Save(draft); err != nil {
		return draft, "", err
	}
	path, err := config.DefaultPath()
	return draft, path, err
}
