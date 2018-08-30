package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	method  = flag.String("X", "HEAD", "HTTP method")
	urlKey  = flag.String("j", "url", "key under which to find the URLs to check")
	verbose = flag.Bool("verbose", false, "verbose output")
)

// Result of a link check.
type Result struct {
	Link       string      `json:"link"`
	StatusCode int         `json:"status"`
	T          time.Time   `json:"t"`
	Comment    string      `json:"comment"`
	Payload    interface{} `json:"payload"`
}

func main() {
	flag.Parse()

	br := bufio.NewReader(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	for {
		line, err := br.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var payload = make(map[string]interface{})
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			log.Fatal(err)
		}
		value, ok := payload[*urlKey]
		if !ok {
			continue
		}

		var links []string

		switch t := value.(type) {
		case string:
			links = append(links, t)
		case []string:
			for _, v := range t {
				links = append(links, v)
			}
		case []interface{}:
			for _, v := range t {
				links = append(links, fmt.Sprintf("%v", v))
			}
		default:
			log.Printf("ignoring %T", t)
			continue
		}

		if *verbose {
			for _, v := range links {
				log.Println(v)
			}
		}

		client := http.DefaultClient
		var results []Result

		for _, v := range links {
			req, err := http.NewRequest(*method, v, nil)
			if err != nil {
				result := Result{
					Link:    v,
					T:       time.Now(),
					Comment: err.Error(),
					Payload: payload,
				}
				results = append(results, result)
				log.Printf("failed to create request: %v", err)
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				result := Result{
					Link:    v,
					T:       time.Now(),
					Comment: err.Error(),
					Payload: payload,
				}
				results = append(results, result)
				log.Printf("request failed: %v", err)
				continue
			}
			defer resp.Body.Close()
			result := Result{
				Link:       v,
				StatusCode: resp.StatusCode,
				T:          time.Now(),
				Payload:    payload,
				Comment:    fmt.Sprintf("%s", *method),
			}
			results = append(results, result)
		}

		for _, r := range results {
			if err := enc.Encode(r); err != nil {
				log.Fatal(err)
			}
		}
	}
}
