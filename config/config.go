package config

import (
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/syepes/ping_exporter/pkg/common"
	yaml "gopkg.in/yaml.v3"
)

// Config represents configuration for the exporter
type Config struct {
	Conf struct {
		Refresh    duration `yaml:"refresh"`
		Nameserver string   `yaml:"nameserver"`
	} `yaml:"conf"`
	ICMP struct {
		Interval duration `yaml:"interval"`
		Timeout  duration `yaml:"timeout"`
		Count    int      `yaml:"count"`
	} `yaml:"icmp"`
	MTR struct {
		Interval duration `yaml:"interval"`
		Timeout  duration `yaml:"timeout"`
		MaxHops  int      `yaml:"max-hops"`
		Count    int      `yaml:"count"`
	} `yaml:"mtr"`
	TCP struct {
		Interval duration `yaml:"interval"`
		Timeout  duration `yaml:"timeout"`
	} `yaml:"tcp"`
	Targets []struct {
		Name  string   `yaml:"name"`
		Host  string   `yaml:"host"`
		Type  string   `yaml:"type"`
		Probe []string `yaml:"probe"`
	} `yaml:"targets"`
}

type duration time.Duration

// SafeConfig Safe configuration reload
type SafeConfig struct {
	Cfg *Config
	sync.RWMutex
}

// ReloadConfig Safe configuration reload
func (sc *SafeConfig) ReloadConfig(confFile string) (err error) {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	var c = &Config{}
	f, err := os.Open(confFile)
	if err != nil {
		return fmt.Errorf("Reading config file: %s", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	if err = decoder.Decode(c); err != nil {
		return fmt.Errorf("Parsing config file: %s", err)
	}

	// Validate and Filter config
	targets := c.Targets[:0]
	var targetNames []string

	for _, t := range c.Targets {
		targetNames = append(targetNames, t.Name)
		found, _ := regexp.MatchString("^ICMP|MTR|ICMP+MTR|TCP$", t.Type)
		if found == false {
			return fmt.Errorf("Target '%s' has unknown check type '%s' must be one of (ICMP|MTR|ICMP+MTR|TCP)", t.Name, t.Type)
		}

		// Filter out the targets that are not assigned to the running host, if the `probe` is not specified don't filter
		if t.Probe == nil {
			targets = append(targets, t)
		} else {
			for _, p := range t.Probe {
				if p == hostname {
					targets = append(targets, t)
					continue
				}
			}
		}
	}

	// Remap the filtered targets
	c.Targets = targets

	if _, err = common.HasListDuplicates(targetNames); err != nil {
		return fmt.Errorf("Parsing config file: %s", err)
	}

	// Config precheck
	if c.MTR.MaxHops < 0 || c.MTR.MaxHops > 65500 {
		return fmt.Errorf("mtr.max-hops must be between 0 and 65500")
	}
	if c.MTR.Count < 0 || c.MTR.Count > 65500 {
		return fmt.Errorf("mtr.count must be between 0 and 65500")
	}

	sc.Lock()
	sc.Cfg = c
	sc.Unlock()

	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (d *duration) UnmarshalYAML(unmashal func(interface{}) error) error {
	var s string
	if err := unmashal(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = duration(dur)
	return nil
}

// Duration is a convenience getter.
func (d duration) Duration() time.Duration {
	return time.Duration(d)
}

// Set updates the underlying duration.
func (d *duration) Set(dur time.Duration) {
	*d = duration(dur)
}
