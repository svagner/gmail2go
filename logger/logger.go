package logger

import (
	"log"
	"os"
)

var (
	debug   bool
	logFile os.File
)

func Init(dbg bool, filename string) error {
	if filename != "" {
		logFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		log.SetOutput(logFile)
	}
	debug = dbg
	return nil
}

func DebugPrint(text ...interface{}) {
	if debug {
		log.Println("[DEBUG] ", text)
	}
}

func InfoPrint(text ...interface{}) {
	log.Println("[INFO] ", text)
}

func WarningPrint(text ...interface{}) {
	log.Println("[WARNING] ", text)
}

func CriticalPrint(text ...interface{}) {
	log.Fatalln("[Critical] ", text)
}

func Delete() {
	logFile.Close()
}
