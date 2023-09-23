package dlcleaner

import (
	_ "embed"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"time"

	"github.com/hairyhenderson/go-which"
	cp "github.com/otiai10/copy"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

//go:embed sample_config.yaml
var sampleConfFile string

type Job struct {
	Name     string `yaml:"name"`
	Src      string `yaml:"src"`
	Dst      string `yaml:"dst"`
	Schedule string `yaml:"schedule"`
}

type JobConfig struct {
	Jobs []Job `yaml:"jobs"`
}

type CachedJob struct {
	Name    string    `yaml:"name"`
	NextRun time.Time `yaml:"next_run"`
}
type JobCache struct {
	Jobs []CachedJob `yaml:"jobs"`
}

type Paths struct {
	ConfPath  string
	CachePath string
}

func GetPaths() (Paths, error) {
	usr, err := user.Current()
	if err != nil {
		return Paths{}, err
	}

	confPath := filepath.Join(usr.HomeDir, ".config", "dlcleaner", "config.yaml")
	cachePath := filepath.Join(usr.HomeDir, ".config", "dlcleaner", "cache.yaml")

	return Paths{
		ConfPath:  confPath,
		CachePath: cachePath,
	}, nil
}

func BinCheck(bins []string) error {
	for _, bin := range bins {
		if !which.Found(bin) {
			return fmt.Errorf("command: '%s' not found", bin)
		}
	}
	return nil
}

func LoadConfig(confPath string) (JobConfig, error) {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return JobConfig{}, err
	}
	jobConfig := JobConfig{}
	err = yaml.Unmarshal(data, &jobConfig)
	if err != nil {
		return jobConfig, err
	}

	return jobConfig, nil
}

func LoadCache(cachePath string) (JobCache, error) {
	data, err := os.ReadFile(cachePath)
	if os.IsNotExist(err) {
		return JobCache{}, nil
	} else if err != nil {
		return JobCache{}, err
	}
	jobCache := JobCache{}
	err = yaml.Unmarshal(data, &jobCache)
	if err != nil {
		return jobCache, err
	}

	return jobCache, nil
}

func WriteNewConfFile(confPath string) error {
	dir, _ := filepath.Split(confPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	err := os.WriteFile(confPath, []byte(sampleConfFile), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func WriteCacheFile(cachePath string, cache JobCache) error {
	dir, _ := filepath.Split(cachePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	data, err := yaml.Marshal(&cache)
	if err != nil {
		return err
	}

	err = os.WriteFile(cachePath, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func FindNextRunFromCache(jobName string, cache JobCache) (bool, time.Time) {
	for _, job := range cache.Jobs {
		if job.Name == jobName {
			return true, job.NextRun
		}
	}
	return false, time.Time{}
}

func getNextRun(cronExpr string) (time.Time, error) {
	schedule, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return time.Time{}, err
	}
	now := time.Now()

	return schedule.Next(now), nil
}

func UpdateNextRun(job Job, jobCache *JobCache) error {
	for i, eachJob := range jobCache.Jobs {
		if eachJob.Name == job.Name {
			nextRun, err := getNextRun(job.Schedule)
			if err != nil {
				return err
			}
			jobCache.Jobs[i].NextRun = nextRun
			return nil
		}
	}

	nextRun, err := getNextRun(job.Schedule)
	if err != nil {
		return err
	}

	jobCache.Jobs = append(jobCache.Jobs, CachedJob{
		job.Name,
		nextRun,
	})
	return nil
}

func getTimestamp() string {
	format := "2006-01-02-15-04-05"
	t := time.Now()
	timestamp := t.Format(format)
	return timestamp
}

func containsFiles(dir string) (bool, error) {
	var contains bool
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && path != dir {
			contains = true
			return fmt.Errorf("file found")
		}
		return nil
	})

	if err != nil && err.Error() == "file found" {
		return contains, nil
	}
	return contains, err
}

func removeAllContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		} else {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	return nil
}

func RunJob(job Job) error {
	_, err := os.Stat(job.Src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(job.Dst, os.ModePerm); err != nil {
		return err
	}

	contains, err := containsFiles(job.Src)
	if err != nil {
		return err
	}
	if !contains {
		return nil
	}

	fmt.Printf("Copying... %s --> %s\n", job.Src, path.Join(job.Dst, getTimestamp()))
	cp.Copy(job.Src, path.Join(job.Dst, getTimestamp()))

	err = removeAllContents(job.Src)
	if err != nil {
		return err
	}

	return nil
}
