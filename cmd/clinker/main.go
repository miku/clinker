package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/miku/clinker/xflag"
	"github.com/sethgrid/pester"
	log "github.com/sirupsen/logrus"
)

var Version = "0.2.6"

var (
	method        = flag.String("X", "GET", "HTTP method")
	urlKey        = flag.String("j", "url", "key under which to find the URLs to check")
	verbose       = flag.Bool("verbose", false, "verbose output")
	numWorkers    = flag.Int("w", runtime.NumCPU(), "number of workers")
	bestEffort    = flag.Bool("b", false, "skip invalid input")
	batchSize     = flag.Int("size", 100, "batch urls")
	showVersion   = flag.Bool("version", false, "show version")
	userAgent     = flag.String("ua", fmt.Sprintf("clinker/%s (https://git.io/fAC27)", Version), "use a specific user agent")
	headerProfile = flag.String("hp", "", "use additional header profile")
)

var headerProfiles = map[string]map[string]string{
	"basic": map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.5",
		"Accept-Encoding": "gzip, deflate, br",
		"DNT":             "1",
	},
}

func prependSchema(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return fmt.Sprintf("http://%s", s)
}

// RedirectRecorder records intermediate requests.
type RedirectRecorder struct {
	Reqs []*http.Request
}

func (rr *RedirectRecorder) Record(req *http.Request, via []*http.Request) error {
	rr.Reqs = via
	return nil
}

func (rr *RedirectRecorder) Reset() {
	rr.Reqs = nil
}

type RedirectEntry struct {
	Status int    `json:"status"`
	URL    string `json:"url"`
}

func (rr *RedirectRecorder) Entries() (entries []*RedirectEntry) {
	for i, r := range rr.Reqs {
		if i == 0 {
			continue
		}
		entry := &RedirectEntry{
			URL:    r.URL.String(),
			Status: r.Response.StatusCode,
		}
		entries = append(entries, entry)
	}
	return entries
}

// worker is a vanilla worker working on batches of lines. Each line can result
// in zero, one or more links to be checked.
func worker(queue chan []string, headers http.Header, resultc chan []Result, wg *sync.WaitGroup) {
	defer wg.Done()

	redirectRecorder := RedirectRecorder{}
	// Use extended client, so we can skip certificate validation.
	client := pester.NewExtendedClient(&http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		CheckRedirect: redirectRecorder.Record,
	})
	client.Concurrency = 3
	client.MaxRetries = 5
	client.Backoff = pester.ExponentialBackoff
	client.KeepLog = false
	client.Timeout = 30 * time.Second
	client.SetRetryOnHTTP429(true)

	var started time.Time

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
				for k, vs := range headers {
					for _, v := range vs {
						req.Header.Add(k, v)
					}
				}
				started = time.Now()
				resp, err := client.Do(req)
				if err != nil {
					result := Result{
						Link:    v,
						T:       time.Now(),
						Comment: err.Error(),
						Payload: payload,
						Elapsed: time.Since(started),
					}
					results = append(results, result)
					log.Printf("request failed: %v", err)
					continue
				}
				if err := resp.Body.Close(); err != nil {
					log.Println(err)
				}
				result := Result{
					RequestHeaders: headers,
					Link:           v,
					StatusCode:     resp.StatusCode,
					T:              time.Now(),
					Payload:        payload,
					Comment:        fmt.Sprintf("%s", *method),
					Headers:        resp.Header,
					Elapsed:        time.Since(started),
					Redirects:      redirectRecorder.Entries(),
				}
				results = append(results, result)
				redirectRecorder.Reset()
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
	Link           string           `json:"link,omitempty"`
	RequestHeaders http.Header      `json:"h,omitempty"`
	StatusCode     int              `json:"status,omitempty"`
	T              time.Time        `json:"t,omitempty"`
	Elapsed        time.Duration    `json:"elapsed,omitempty"`
	Comment        string           `json:"comment,omitempty"`
	Payload        interface{}      `json:"payload,omitempty"`
	Headers        http.Header      `json:"headers,omitempty"`
	Redirects      []*RedirectEntry `json:"redirects,omitempty"`
}

func main() {
	var headerFlags xflag.ArrayFlags
	flag.Var(&headerFlags, "H", "HTTP header to send (repeatable)")

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

	var headers = make(http.Header)
	for _, hf := range headerFlags {
		parts := strings.SplitN(hf, ":", 2)
		if len(parts) != 2 {
			log.Fatal("header must be in key:value format, not %s", hf)
		}
		headers.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}
	if profile, ok := headerProfiles[*headerProfile]; ok {
		for k, v := range profile {
			headers.Add(k, v)
		}
	}
	headers.Add("User-Agent", *userAgent)

	var wg sync.WaitGroup

	for i := 0; i < *numWorkers; i++ {
		wg.Add(1)
		go worker(queue, headers, resultc, &wg)
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
		line = strings.TrimSpace(line)

		// Allows plain URL list of be handled as well.
		if !strings.HasPrefix(line, "{") {
			line = fmt.Sprintf(`{"url": %q}`, prependSchema(line))
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
