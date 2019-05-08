package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %+v\n", err)
	}
}

// run could be split into multiple functions by each step,
// but leaving it now as it is since it is not that long.
func run() error {
	// Command line parsing
	flagOut := flag.String("out", "", "output file to write stats to")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Printf("Usage: %s <FLAGS> GWI_API_BASE_URL\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}
	addr := strings.TrimRight(flag.Arg(0), "/")

	// Consult the API.
	u := addr + "/v1/stats/teams/member-count"
	resp, err := http.Get(u)
	if err != nil {
		return errors.Wrap(err, "failed to call the API")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected status code returned: %d", resp.StatusCode)
	}

	// Decode the response body.
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return errors.Wrap(err, "failed to decode response body")
	}
	resp.Body.Close()

	// Prepare the output file.
	var output io.Writer
	if p := *flagOut; p != "" {
		fd, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)
		if err != nil {
			return errors.Wrap(err, "failed to open output file")
		}
		defer fd.Close()
		output = fd
	} else {
		output = os.Stdout
	}

	// Encode the output line.
	stats := map[string]interface{}{
		"teams.member_count": body,
	}
	return errors.Wrap(json.NewEncoder(output).Encode(stats), "failed to write stats")
}
