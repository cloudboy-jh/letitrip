package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/cloudboy-jh/letitrip/letitrip/internal/config"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/disc"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/metadata"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/organize"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/ui"
)

func Execute() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "config" {
		runConfig(args[1:])
		return
	}

	fs := flag.NewFlagSet("letitrip", flag.ExitOnError)
	output := fs.String("output", "", "Output directory")
	yes := fs.Bool("yes", false, "Skip confirmation prompts")
	info := fs.Bool("info", false, "Show disc info without ripping")
	plain := fs.Bool("plain", false, "Disable the TUI dashboard")
	cpuProfile := fs.String("cpuprofile", "", "Write CPU profile to file")
	memProfile := fs.String("memprofile", "", "Write heap profile to file")
	fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(1)
	}

	if *output != "" {
		cfg.OutputDir = *output
	} else {
		defaultOutput := cfg.OutputDir
		if defaultOutput == "" {
			defaultOutput = config.DefaultOutputDir()
		}
		if !*yes && !*info && term.IsTerminal(int(os.Stdin.Fd())) {
			selected, err := promptOutputDir(defaultOutput)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Failed to read output directory:", err)
				os.Exit(1)
			}
			if selected != "" {
				cfg.OutputDir = selected
			} else {
				cfg.OutputDir = defaultOutput
			}
		} else {
			cfg.OutputDir = defaultOutput
		}
	}

	if *cpuProfile != "" {
		if err := startCPUProfile(*cpuProfile); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to start CPU profile:", err)
			os.Exit(1)
		}
		defer stopCPUProfile()
	}

	if *memProfile != "" {
		defer func() {
			if err := writeHeapProfile(*memProfile); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to write heap profile:", err)
			}
		}()
	}

	drive := cfg.Drive
	if drive == "" {
		drive, err = disc.DetectDrive()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to detect CD drive:", err)
			os.Exit(1)
		}
	}

	toc, err := disc.ReadTOC(drive)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read disc:", err)
		os.Exit(1)
	}

	discID, err := disc.CalculateDiscID(toc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to calculate disc ID:", err)
		os.Exit(1)
	}

	meta, err := metadata.FetchDiscMetadata(discID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Metadata lookup failed:", err)
		meta = metadata.FallbackMetadata(toc)
	}

	if *info {
		printInfo(meta, discID)
		return
	}

	printInfo(meta, discID)
	if !*yes {
		if !confirm("Proceed with ripping?") {
			fmt.Fprintln(os.Stdout, "Aborted.")
			return
		}
	}

	useTUI := !*plain && term.IsTerminal(int(os.Stdout.Fd()))
	if useTUI {
		events := make(chan organize.ProgressEvent, 100)
		reporter := organize.NewChannelReporter(events)
		errCh := make(chan error, 1)

		go func() {
			errCh <- organize.RipAndOrganize(drive, cfg, toc, meta, reporter)
			close(events)
		}()

		program := tea.NewProgram(ui.NewModel(meta, discID, events))
		if err := program.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "TUI failed:", err)
		}

		if err := <-errCh; err != nil {
			fmt.Fprintln(os.Stderr, "Ripping failed:", err)
			os.Exit(1)
		}
	} else {
		reporter := organize.NewCLIReporter(os.Stdout, os.Stderr)
		if err := organize.RipAndOrganize(drive, cfg, toc, meta, reporter); err != nil {
			fmt.Fprintln(os.Stderr, "Ripping failed:", err)
			os.Exit(1)
		}
	}

	if cfg.EjectOnComplete {
		if err := disc.Eject(drive); err != nil {
			fmt.Fprintln(os.Stderr, "Eject failed:", err)
		}
	}
}

func runConfig(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(1)
	}

	fs := flag.NewFlagSet("config", flag.ExitOnError)
	output := fs.String("output", "", "Default output directory")
	drive := fs.String("drive", "", "Default CD drive letter (e.g. D:)")
	eject := fs.Bool("eject", cfg.EjectOnComplete, "Eject disc after rip")
	fs.Parse(args)

	if *output != "" {
		cfg.OutputDir = *output
	}

	if *drive != "" {
		cfg.Drive = *drive
	}

	cfg.EjectOnComplete = *eject

	if err := config.Save(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save config:", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, "Configuration saved.")
}

func confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stdout, "%s [y/N]: ", prompt)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func promptOutputDir(defaultOutput string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stdout, "Output directory [%s]: ", defaultOutput)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	response = strings.TrimSpace(response)
	if response == "" {
		return "", nil
	}
	if strings.HasPrefix(response, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			remainder := strings.TrimPrefix(response, "~")
			remainder = strings.TrimPrefix(remainder, string(os.PathSeparator))
			remainder = strings.TrimPrefix(remainder, "/")
			response = filepath.Join(home, remainder)
		}
	}
	if !filepath.IsAbs(response) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		response = filepath.Join(cwd, response)
	}
	return filepath.Clean(response), nil
}

func printInfo(meta metadata.Release, discID string) {
	fmt.Fprintf(os.Stdout, "Disc ID: %s\n", discID)
	fmt.Fprintf(os.Stdout, "Artist: %s\n", meta.Artist)
	fmt.Fprintf(os.Stdout, "Album: %s\n", meta.Album)
	if meta.Year != "" {
		fmt.Fprintf(os.Stdout, "Year: %s\n", meta.Year)
	}
	fmt.Fprintf(os.Stdout, "Tracks: %d\n", len(meta.Tracks))
	for _, track := range meta.Tracks {
		fmt.Fprintf(os.Stdout, "  %02d - %s\n", track.Number, track.Title)
	}
}

var cpuProfileFile *os.File

func startCPUProfile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		_ = file.Close()
		return err
	}
	cpuProfileFile = file
	return nil
}

func stopCPUProfile() {
	if cpuProfileFile == nil {
		return
	}
	pprof.StopCPUProfile()
	_ = cpuProfileFile.Close()
	cpuProfileFile = nil
}

func writeHeapProfile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	runtime.GC()
	return pprof.WriteHeapProfile(file)
}
