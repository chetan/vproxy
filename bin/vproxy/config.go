package main

import (
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml"
	"github.com/urfave/cli/v2"
)

// Config file fields for vproxy
type Config struct {
	Verbose bool

	Server struct {
		Verbose bool

		Listen string
		HTTP   int
		HTTPS  int

		CaRootPath string `toml:"caroot_path"`
		CertPath   string `toml:"cert_path"`
	}

	Client struct {
		Verbose bool

		Host string
		HTTP int
		Bind string
	}
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// findConfig locates a config file at the given locations with either a .conf or .toml extension
// (file format must be TOML, however)
func findConfig(files ...string) string {
	for _, config := range files {
		if config != "" {
			if fileExists(config) {
				return config
			}
			if strings.Contains(config, ".conf") {
				// look for .toml also
				conf := strings.Replace(config, ".conf", ".toml", 1)
				if fileExists(conf) {
					return conf
				}
			}
		}
	}
	return ""
}

func homeConfPath() string {
	d, err := os.UserHomeDir()
	if err == nil {
		return path.Join(d, ".vproxy.conf")
	}
	return ""
}

func loadConfigFile(path string) (*Config, error) {
	t, err := toml.LoadFile(path)
	if err != nil {
		return nil, err
	}
	var conf Config
	err = t.Unmarshal(&conf)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}

// transform listen addr arg
func cleanListenAddr(c *cli.Context) {
	listen := c.String("listen")
	if listen == "" {
		c.Set("listen", listenDefaultAddr)
	} else if listen == "0" {
		c.Set("listen", listenAnyIP)
	}
}

func loadClientConfig(c *cli.Context) error {
	conf := findConfigFile(c.String("config"), false)
	return loadConfig(c, conf)
}

func loadDaemonConfig(c *cli.Context) error {
	conf := findConfigFile(c.String("config"), true)
	return loadConfig(c, conf)
}

func loadConfig(c *cli.Context, conf string) error {
	if c.IsSet("config") {
		if cf := c.String("config"); conf != cf {
			// config flag was passed but file does not exist
			log.Fatalf("error: config file not found: %s\n", cf)
		}
	}
	if conf == "" {
		return nil
	}

	verbose(c, "Loading config file %s", conf)
	config, err := loadConfigFile(conf)
	if err != nil {
		return err
	}

	if config != nil {
		if v := (config.Server.Verbose || config.Verbose); v && !c.IsSet("verbose") {
			c.Lineage()[1].Set("verbose", "true")
			verbose(c, "Loading config file %s", conf)
			verbose(c, "via conf: verbose=true")
		}
		if v := config.Server.Listen; v != "" && !c.IsSet("listen") {
			verbose(c, "via conf: listen=%s", v)
			c.Set("listen", v)
		}
		if v := config.Server.HTTP; v > 0 && !c.IsSet("http") {
			verbose(c, "via conf: http=%d", v)
			c.Set("http", strconv.Itoa(v))
		}
		if v := config.Server.HTTPS; v > 0 && !c.IsSet("https") {
			verbose(c, "via conf: https=%d", v)
			c.Set("https", strconv.Itoa(v))
		}
		if v := config.Server.CaRootPath; v != "" {
			os.Setenv("CAROOT_PATH", v)
			verbose(c, "via conf: CAROOT_PATH=%s", v)
		}
		if v := config.Server.CertPath; v != "" {
			os.Setenv("CERT_PATH", v)
			verbose(c, "via conf: CERT_PATH=%s", v)
		}

		// client configs
		if v := (config.Client.Verbose || config.Verbose); v && !c.IsSet("verbose") {
			c.Lineage()[1].Set("verbose", "true")
			verbose(c, "Loading config file %s", conf)
			verbose(c, "via conf: verbose=true")
		}
		if v := config.Client.Host; v != "" && !c.IsSet("host") {
			verbose(c, "via conf: host=%s", v)
			c.Set("host", v)
		}
		if v := config.Client.HTTP; v > 0 && !c.IsSet("http") {
			verbose(c, "via conf: http=%d", v)
			c.Set("http", strconv.Itoa(v))
		}
		if v := config.Client.Bind; v != "" && !c.IsSet("bind") {
			verbose(c, "via conf: bind=%s", v)
			c.Set("bind", v)
		}
		if v := config.Server.CaRootPath; v != "" {
			os.Setenv("CAROOT_PATH", v)
			verbose(c, "via conf: CAROOT_PATH=%s", v)
		}
	}
	cleanListenAddr(c)
	return nil
}
