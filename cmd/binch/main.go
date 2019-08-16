package main

import (
	"github.com/tunz/binch-go/pkg/io"
	"github.com/tunz/binch-go/pkg/view"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"log"
	"os"
)

var filename = kingpin.Arg("file", "ELF binary to edit.").Required().String()
var logfile = kingpin.Flag("log", "Log filename.").Default(os.DevNull).String()

func setupLogfile(logfile string) *os.File {
	fpLog, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	log.SetOutput(fpLog)
	return fpLog
}

func main() {
	kingpin.Parse()

	var logfp *os.File
	setupLogfile(*logfile)
	defer logfp.Close()

	binary := bcio.ReadElf(*filename)
	bcview.Run(*filename, binary)
}
