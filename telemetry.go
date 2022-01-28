package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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

	cmd := exec.Command("sh", "-c", command)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fatal(err)
	}

	if err := cmd.Start(); err != nil {
		fatal(err)
	}

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
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintln(os.Stderr, scanner.Text())
		}
	}()
}
