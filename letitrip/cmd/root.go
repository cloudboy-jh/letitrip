package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/cloudboy-jh/letitrip/letitrip/internal/config"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/disc"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/metadata"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/organize"
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
	fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(1)
	}

	if *output != "" {
		cfg.OutputDir = *output
	}

	if cfg.OutputDir == "" {
		cfg.OutputDir = config.DefaultOutputDir()
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

	if err := organize.RipAndOrganize(drive, cfg, toc, meta); err != nil {
		fmt.Fprintln(os.Stderr, "Ripping failed:", err)
		os.Exit(1)
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
