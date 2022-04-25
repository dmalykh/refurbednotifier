package config

import (
	"flag"
	"fmt"
	"net/url"
	"time"
)

type Config struct {
	URL          string
	Rps          uint64
	Timeout      time.Duration
	Interval     time.Duration
	MaxQueueSize int
}

// ParseFlags returns cancelFunc for display for user and error if it exists. If cantFunc returned, proccess have to exit.
func (c *Config) ParseFlags() (func(), error) {
	var help bool
	flag.StringVar(&c.URL, `url`, ``, `URL for sending requests (required)`)
	flag.Uint64Var(&c.Rps, `rps`, 100, `Requests per second will be sent`)
	flag.DurationVar(&c.Timeout, `timeout`, 5*time.Second, `Timeout for every HTTP requests`)
	flag.DurationVar(&c.Interval, `interval`, 1*time.Second, `Interval for send messages`)
	flag.BoolVar(&help, `help`, false, `Displays this help`)
	flag.Parse()

	if help || flag.NFlag() == 0 {
		return flag.Usage, nil
	}
	if err := c.Validate(); err != nil {
		return flag.Usage, err
	}

	return nil, nil
}

// Validate flags values
func (c *Config) Validate() error {
	{
		u, err := url.Parse(c.URL)
		if err != nil {
			return fmt.Errorf(`"%s" is not valid URL %w`, c.URL, err)
		}
		if u.Host == `` || u.Scheme == `` {
			return fmt.Errorf(`"%s" should have host and scheme`, c.URL)
		}
	}
	{
		if c.Rps == 0 {
			return fmt.Errorf(`"%s" should have host and scheme`, c.URL)
		}
	}

	return nil
}
