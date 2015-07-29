package config

import (
	"io/ioutil"
	"log"
	"os"
	"time"

	"code.google.com/p/gcfg"
)

const (
	defaultContent = `[global]
print = false
colorPrint = false
checkPeriod = 60
passwd = ~/.gmail2go_passwd
`
)

type AccountConfig struct {
	Email        string
	Notify       bool
	Speech       bool
	DictMessages bool
}

type Config struct {
	Global struct {
		Print       bool
		ColorPrint  bool
		CheckPeriod time.Duration
		Passwd      string
		Daemonize   bool
		Logfile     string
		Pidfile     string
		CheckDump   string
		Debug       bool
	}
	Account map[string]*AccountConfig
}

func (self *Config) ParseConfig(file string) error {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		log.Printf("Creating default config file %s", file)
		if err = createDefault(file); err != nil {
			log.Fatalln("Couldn't create config file ", file, err.Error())
		}
	}
	if err := gcfg.ReadFileInto(self, file); err != nil {
		return err
	}
	return nil
}

func createDefault(file string) error {
	err := ioutil.WriteFile(file, []byte(defaultContent), 0700)
	return err
}
