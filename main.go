package main

import (
	"fmt"
	"os"
	"time"

	dlcleaner "github.com/BonyChops/dlcleaner/internal"
)

// const paths.ConfPath = "~/.config/dlcleaner/config.yaml"
// const paths.CachePath = "~/.config/dlcleaner/cache.yaml"

func main() {
	bins := []string{"zip", "rm"}
	if err := dlcleaner.BinCheck(bins); err != nil {
		fmt.Println(err)
		return
	}

	paths, err := dlcleaner.GetPaths()
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = os.Stat(paths.ConfPath)
	if err != nil {
		err = dlcleaner.WriteNewConfFile(paths.ConfPath)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Config file created at: ", paths.ConfPath)
		fmt.Println("Rewrite config file, then register the command 'dlcleaner' to your cron job.")
		return
	}

	conf, err := dlcleaner.LoadConfig(paths.ConfPath)
	if err != nil {
		fmt.Println("Config file error occurred.")
		fmt.Println("Config file path: ", paths.ConfPath)
		fmt.Println(err)
		return
	}

	cache, err := dlcleaner.LoadCache(paths.CachePath)
	if err != nil {
		fmt.Println("Cache file error occurred.")
		fmt.Println("Cache file path: ", paths.CachePath)
		fmt.Println(err)
		return
	}

	for _, job := range conf.Jobs {
		found, nextRun := dlcleaner.FindNextRunFromCache(job.Name, cache)
		if found {
			if time.Now().After(nextRun) {
				fmt.Println("run")
				err = dlcleaner.RunJob(job)
				if err != nil {
					fmt.Println(err)
					return
				}
				dlcleaner.UpdateNextRun(job, &cache)
			} else {
				fmt.Println("skip")
				continue
			}
		} else {
			err = dlcleaner.UpdateNextRun(job, &cache)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
	dlcleaner.WriteCacheFile(paths.CachePath, cache)
}
