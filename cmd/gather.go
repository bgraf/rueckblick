package cmd

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/bgraf/rueckblick/filesystem"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// gatherCmd represents the gather command
var gatherCmd = &cobra.Command{
	Use: "gather",
	// TODO: Short, Long
	RunE: runGather,
}

func init() {
	rootCmd.AddCommand(gatherCmd)

	gatherCmd.Flags().StringP("destination", "d", "", "Destination root")
	gatherCmd.Flags().StringP("from", "f", time.Now().Format("2006-01-02"), "Earliest date to copy")
	gatherCmd.Flags().BoolP("all", "a", false, "Copy all and ignore from date")
}

func runGather(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Empty args, take sources from configuration
		args = append(args, viper.GetStringSlice("gather.sources")...)
	}

	if len(args) == 0 {
		return fmt.Errorf("no sources given and no sources specified in configuration")
	}

	destRoot, err := cmd.Flags().GetString("destination")
	if err != nil {
		return nil
	}

	if destRoot == "" {
		destRoot = viper.GetString("gather.destination")
	}

	if destRoot == "" {
		return fmt.Errorf("destination not specified in arguments or configuration")
	}

	destinationMap, err := newTargetDirectories(destRoot)
	if err != nil {
		return err
	}

	fromDate := time.Time{}
	if all, err := cmd.Flags().GetBool("all"); err != nil || !all {
		// Argument: from date
		dateS, err := cmd.Flags().GetString("from")
		if err != nil {
			return err
		}

		fromDate, err = time.Parse("2006-01-02", dateS)
		if err != nil {
			return fmt.Errorf("parsing from date: %w", err)
		}

		fromDate = time.Date(fromDate.Year(), fromDate.Month(), fromDate.Day(), 0, 0, 0, 0, time.Local)
	}

	for _, root := range args {
		log.Printf("scanning root %s\n", root)
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.Type().IsRegular() {
				return nil
			}

			path = filesystem.Abs(path)
			info, err := d.Info()
			if err != nil {
				return fmt.Errorf("file info: %w", err)
			}

			mod := info.ModTime()
			date := time.Date(mod.Year(), mod.Month(), mod.Day(), 0, 0, 0, 0, time.Local)

			targetDir := destinationMap.Get(date)
			targetFile := filepath.Join(targetDir, d.Name())

			if date.Before(fromDate) {
				log.Printf("skipping '%s', before from date", targetFile)
				return nil
			}

			if filesystem.Exists(targetFile) {
				log.Printf("skipping '%s', already exists", targetFile)
				return nil
			}

			if err := filesystem.CreateDirectoryIfNotExists(targetDir); err != nil {
				return err
			}

			log.Printf("copying to %s\n", targetFile)
			return filesystem.Copy(path, targetFile)
		})

		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

type targetDirectories struct {
	root    string
	mapping map[time.Time]string
}

func (dirs *targetDirectories) Get(t time.Time) string {
	if path, ok := dirs.mapping[t]; ok {
		return path
	}

	return filepath.Join(dirs.root, t.Format("2006-01-02"))
}

var pat = regexp.MustCompile(`^(\d\d\d\d)-(\d\d)-(\d\d)`)

func newTargetDirectories(root string) (dirs *targetDirectories, err error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}

	byDate := make(map[time.Time][]string)
	_ = byDate

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		groups := pat.FindStringSubmatch(name)
		if len(groups) != 4 {
			continue
		}

		y, _ := strconv.Atoi(groups[1])
		m, _ := strconv.Atoi(groups[2])
		d, _ := strconv.Atoi(groups[3])

		date := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.Local)

		byDate[date] = append(byDate[date], name)
	}

	directories := make(map[time.Time]string)

	for date, paths := range byDate {
		if len(paths) == 1 {
			directories[date] = path.Join(root, paths[0])
		} else {
			directories[date] = path.Join(root, date.Format("2006-01-02"))
		}
	}

	dirs = &targetDirectories{
		root:    root,
		mapping: directories,
	}

	return
}
