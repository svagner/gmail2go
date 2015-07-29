package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/svagner/gmail2go/config"
	"github.com/svagner/gmail2go/daemon"
	"github.com/svagner/gmail2go/logger"
	"github.com/svagner/gmail2go/passwd"
	"github.com/svagner/gmail2go/rss"
	"github.com/svagner/gmail2go/sound"
)

var (
	configFile     = flag.String("config", path.Join(os.Getenv("HOME"), ".gmail2gorc"), "the config file")
	accountsMap    passwd.Passwd
	readedMail     = make(map[string]map[string]bool)
	newReadedMails map[string]map[string]bool
)

func main() {
	flag.Parse()
	var conf config.Config
	if err := conf.ParseConfig(*configFile); err != nil {
		logger.CriticalPrint("Error then parse config file: ", err.Error())
	}
	logger.Init(conf.Global.Debug, conf.Global.Logfile)
	accountsMap, err := passwd.NewPasswd(conf.Global.Passwd)
	if err != nil {
		logger.CriticalPrint("Error inialize passwd file: ", err.Error())
	}
	logger.DebugPrint("Check accounts")
	accountsMap.Check(conf.Global.Passwd, conf.Account)

	//
	//	// prepare the colors for display
	//	yellow, green, red, reset := "", "", "", ""
	//	if *color {
	//		yellow, green, red, reset = "\033[1;33m", "\033[0;32m", "\033[0:31m", "\033[0m"
	//	}
	//
	//	emailCountMap := make(map[string]int)
	//	// show accounts
	//	index := 0
	//	for user, _ := range accountsMap {
	//		if index > 0 {
	//			fmt.Print(green + " | ")
	//		}
	//		fmt.Print(yellow + user)
	//		index++
	//	}
	//	fmt.Println(reset + "\n")
	//	// show unread mails

	if conf.Global.CheckDump != "" {
		logger.DebugPrint("Restore messages hash")
		if err := restoreMessagesRead(conf.Global.CheckDump); err != nil {
			logger.WarningPrint("Restore failed:", err)
		}
	}

	if conf.Global.Daemonize {
		logger.InfoPrint("Daemonizing")
		daemon.Daemonize(1, 1)
	}

	logger.DebugPrint("Handle signals")
	//c := make(chan os.Signal, 1)
	//signal.Notify(c, os.Interrupt)
	//signal.Notify(c, syscall.SIGTERM)
	//go func() {

	//	<-c
	//	if conf.Global.CheckDump != "" {
	//		dumpMessagesRead(conf.Global.CheckDump)
	//	}
	//	os.Exit(1)
	//}()

	logger.DebugPrint("Starting ticker")
	tickChan := time.NewTicker(time.Millisecond * conf.Global.CheckPeriod).C

	for {
		logger.DebugPrint("Wait to tick ", conf.Global.CheckPeriod)
		<-tickChan
		newReadedMails = make(map[string]map[string]bool)
		for acc, authData := range accountsMap {
			newReadedMails[acc] = make(map[string]bool)
			logger.DebugPrint("Start rss read...")
			mails, err := rss.Read("https://mail.google.com/mail/feed/atom", authData.UserName, authData.Password)
			logger.DebugPrint("End rss read...")
			if err != nil {
				logger.WarningPrint(authData.UserName+" error: ", err)
				for key, value := range readedMail[acc] {
					newReadedMails[acc][key] = value
				}
				goto REWRITE
			}
			// iterate over mails

			for _, m := range mails {
				newReadedMails[acc][m.Id] = true
				if _, ok := readedMail[acc][m.Id]; ok {
					continue
				}
				if conf.Account[acc].Notify {
					cmd := exec.Command("/usr/bin/notify-send",
						"-i",
						"/usr/share/notify-osd/icons/gnome/scalable/status/notification-message-email.svg",
						"-t",
						"3000",
						"--urgency=critical",
						"-a",
						"gmail2go",
						"["+m.Modified+"] new mail from "+m.Author.Name,
						m.Title)
					cmd.Run()
				}
				if conf.Global.Print {
					fmt.Println("["+authData.UserName+"] ", m.Author, ": ", m.Title, "\n\t\t", m.Summary)
					logger.DebugPrint("["+authData.UserName+"] ", m.Author, ": ", m.Title, "\n\t\t", m.Summary)
				}
				if conf.Account[acc].Speech {
					uri := "http://translate.google.com/translate_tts?ie=UTF-8&tl=en-us&q=" + url.QueryEscape("You have a new mail from "+m.Author.Name+". "+m.Title)
					logger.DebugPrint("Speech", uri)
					sound.Speech(uri)
				}
			}
		REWRITE:
			readedMail[acc] = make(map[string]bool)
			for key, value := range newReadedMails[acc] {
				readedMail[acc][key] = value
			}
		}

	}
	os.Exit(1)
}

func isPressent(mp map[string]int, searched string) bool {
	for k := range mp {
		if k == searched {
			return true
		}
	}
	return false
}

func getKeyList(mp map[string]int) (keys []string) {
	for k := range mp {
		keys = append(keys, k)
	}
	return
}

func dumpMessagesRead(dump string) error {
	dumpMap := make(map[string]map[string]bool)
	for key, value := range readedMail {
		dumpMap[key] = make(map[string]bool)
		for subkey, subvalue := range value {
			dumpMap[key][subkey] = subvalue
		}
	}
	for key, value := range newReadedMails {
		if _, ok := dumpMap[key]; !ok {
			dumpMap[key] = make(map[string]bool)
		}
		for subkey, subvalue := range value {
			dumpMap[key][subkey] = subvalue
		}
	}
	fin, err := os.Create(dump)
	if err != nil {
		return err
	}
	defer fin.Close()
	w := bufio.NewWriter(fin)
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	err = enc.Encode(readedMail)
	w.Write(out.Bytes())
	if err != nil {
		return err
	}
	w.Flush()
	return nil
}

func restoreMessagesRead(dump string) error {
	fin, err := os.Open(dump)
	if err != nil {
		return err
	}
	// if the config file was opened decrypt it and unmarshal accounts map
	r := bufio.NewReader(fin)
	dec := json.NewDecoder(r)
	if err := dec.Decode(&readedMail); err != nil {
		return err
	}
	logger.DebugPrint("Restore backup of readed mails:", readedMail)
	return nil
}
