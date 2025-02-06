package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/parquet-go/parquet-go"
)

type LogRow struct {
	Id       string `json:"id"`
	Pid      string `json:"pid"`
	TimeUnix int64  `json:"time"`
	Msg      string `json:"message"`
	Level    string `json:"level"`
	Time     time.Time
}

type LogWriter struct {
	writer *parquet.GenericWriter[LogRow]
	buf    [1024]LogRow
	i      int
}

func (lw *LogWriter) Write(b []byte) (n int, err error) {
	logs := bytes.Split(b, []byte{'\n'})
	for _, l := range logs {
		if len(l) == 0 {
			continue
		}
		log.Printf("raw json: %s\n", l)
		lr := LogRow{}
		err = json.Unmarshal(l, &lr)
		lr.Time = time.Unix(0, int64(lr.TimeUnix))
		if err != nil {
			log.Fatalf("error unmarshalling: %v\n", err)
		}
		log.Printf("pid: %v\n", lr.Pid)
		if lw.i == 1024 {
			n, err := lw.writer.Write(lw.buf[:])
			if err != nil {
				log.Fatalf("error writing parquet: %v\n", err)
			} else {
				log.Printf("wrote this many: %d\n", n)
			}
			clear(lw.buf[:])
			lw.i = 0
		}
		lw.buf[lw.i] = lr
		lw.i++
	}
	n = len(b)
	return
}
func (lw *LogWriter) Close() error {
	n, err := lw.writer.Write(lw.buf[:lw.i])
	if err != nil {
		log.Fatalf("error writing in close: %v\n", err)
	} else {
		log.Printf("wrote this many in close: %d\n", n)
	}
	err = lw.writer.Close()
	if err != nil {
		log.Fatalf("error closing: %v\n", err)
	}
	return nil
}

func main() {
	logFile, err := os.Create("logs.parquet")
	if err != nil {
		log.Fatalf("failed to create log file: %v\n", err)
	}
	logWriter := LogWriter{
		writer: parquet.NewGenericWriter[LogRow](logFile),
	}
	sim := exec.Command("../sim/sim")
	sim.Stdout = &logWriter
	err = sim.Run()
	if err != nil {
		log.Fatalf("error running sim: %v\n", err)
	}
	logWriter.Close()
}
