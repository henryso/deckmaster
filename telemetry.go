package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Channel for making telemetry requests.
var Telemetry = make(chan interface{})

// Telemetry write request.
type TelemetryWrite struct {
	data map[string]string
}

// Telemetry read request.
type TelemetryRead struct {
	data     map[string]string
	receiver chan bool
}

func readTelemetry(command string) {
	go func() {
		data := make(map[string]string)

		for {
			switch v := (<-Telemetry).(type) {
			case TelemetryWrite:
				for key, value := range v.data {
					data[key] = value
				}
			case TelemetryRead:
				changed := false
				for key, value := range v.data {
					newValue := data[key]
					if newValue != value {
						v.data[key] = newValue
						changed = true
					}
				}
				v.receiver <- changed
			}
		}
	}()

	for {
		start := time.Now()

		fmt.Fprintf(os.Stderr, "Starting telemetry command: %v\n", command)

		var err error
		var stdout, stderr io.ReadCloser
		cmd := exec.Command("sh", "-c", command)
		stdout, err = cmd.StdoutPipe()
		if err == nil {
			stderr, err = cmd.StderrPipe()
		}
		if err == nil {
			err = cmd.Start()
		}
		if err == nil {
			wg := sync.WaitGroup{}
			wg.Add(2)

			go func() {
				decoder := json.NewDecoder(stdout)
				for {
					var data map[string]string
					if err := decoder.Decode(&data); err == io.EOF {
						break
					} else if err != nil {
						fmt.Fprintln(os.Stderr, err)
					} else {
						Telemetry <- TelemetryWrite{data: data}
					}
				}
				wg.Done()
			}()

			go func() {
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					fmt.Fprintln(os.Stderr, scanner.Text())
				}
				wg.Done()
			}()

			wg.Wait()
			err = cmd.Wait()
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Telemetry command failed: %v\n", err)
		}

		if time.Since(start) < (2 * time.Minute) {
			fmt.Fprintln(os.Stderr, "Command ended too quickly; not restarting")
			break
		} else {
			fmt.Fprintln(os.Stderr, "Command ended; restarting")
		}
	}
}
