package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var Version = "0.1.0"

var (
	method      = flag.String("X", "GET", "HTTP method")
	urlKey      = flag.String("j", "url", "key under which to find the URLs to check")
	verbose     = flag.Bool("verbose", false, "verbose output")
	numWorkers  = flag.Int("w", runtime.NumCPU(), "number of workers")
	bestEffort  = flag.Bool("b", false, "skip invalid input")
	batchSize   = flag.Int("size", 100, "batch urls")
	showVersion = flag.Bool("version", false, "show version")
)

// worker is a vanilla worker working on batches of lines. Each line can result
// in zero, one or more links to be checked.
func worker(queue chan []string, resultc chan []Result, wg *sync.WaitGroup) {
	defer wg.Done()
	client := http.DefaultClient

	for batch := range queue {
		for _, line := range batch {

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			var payload = make(map[string]interface{})
			if err := json.Unmarshal([]byte(line), &payload); err != nil {
				if !*bestEffort {
					log.Fatal(err)
				}
				log.Println(err)
				continue
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

			var results []Result

			for _, v := range links {
				// XXX: GET with something like SectionReader?
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
				req.Header.Set("User-Agent", fmt.Sprintf("clinker/%s (https://git.io/fAC27)", Version))
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
				if err := resp.Body.Close(); err != nil {
					log.Println(err)
				}
				result := Result{
					Link:       v,
					StatusCode: resp.StatusCode,
					T:          time.Now(),
					Payload:    payload,
					Comment:    fmt.Sprintf("%s", *method),
					Header:     resp.Header,
				}
				results = append(results, result)
			}
			resultc <- results
		}
	}
}

// writer writes results to a given writer.
func writer(w io.Writer, resultc chan []Result, done chan bool) {
	enc := json.NewEncoder(w)
	for batch := range resultc {
		for _, result := range batch {
			enc.Encode(result)
		}
	}
	done <- true
}

// Result of a link check.
type Result struct {
	Link       string      `json:"link,omitempty"`
	StatusCode int         `json:"status,omitempty"`
	T          time.Time   `json:"t,omitempty"`
	Comment    string      `json:"comment,omitempty"`
	Payload    interface{} `json:"payload,omitempty"`
	Header     http.Header `json:"header,omitempty"`
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	br := bufio.NewReader(os.Stdin)
	bw := bufio.NewWriter(os.Stdout)
	defer bw.Flush()

	queue := make(chan []string)
	resultc := make(chan []Result)
	done := make(chan bool)

	var wg sync.WaitGroup

	for i := 0; i < *numWorkers; i++ {
		wg.Add(1)
		go worker(queue, resultc, &wg)
	}
	go writer(bw, resultc, done)

	var batch []string

	for {
		line, err := br.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if len(batch) == *batchSize {
			// XXX: required?
			b := make([]string, *batchSize)
			copy(b, batch)
			queue <- b
			batch = nil
		}
		batch = append(batch, line)
	}

	queue <- batch

	close(queue)
	wg.Wait()
	close(resultc)
	<-done
}
