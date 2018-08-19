package main

import (
    "fmt"
    "sync"
    "time"
    "bufio"
    "strings"
    "net/http"
    "io/ioutil"
)

/*
 * Pipeline:
 * First  Stage : Scanner from cmd arg URLs separated by \n;
 * Second stage : Get URL's content from WEB and parse it.
 *
 * At the end sum number of total results (just atomic).
 */

const (
    TheWord = "Go"
    MaxContentGrabbers = 3
)

type Result struct {
    Url   string
    Count int
}

func main() {
    urls := "https://blog.golang.org/pipelines/\nhttps://habr.com/company/mailru/blog/314804/\nhttps://gobyexample.com/atomic-counters/\nhttp://biasedbit.com/blog/golang-custom-transports/"
    // For total value
    var total int
    // First stage
    prod := produce(urls)

    grabbers := make([]<-chan Result, MaxContentGrabbers)
    for i := 0; i < MaxContentGrabbers; i++ {
        // Second stage
        grabbers[i] = parseContent(prod)
    }

    var sb strings.Builder
    for out := range merge(grabbers) {
        sb.WriteString(fmt.Sprintf("[%s]\t\t->\t[%d]\n", out.Url, out.Count))
        total += out.Count
    }

    fmt.Printf("=============\n")
    fmt.Printf("%s", sb.String())
    fmt.Printf("Total is [%d]\n", total)
    fmt.Printf("=============\n")
}

func produce(urls string) (out chan string) {
    out = make(chan string)
    go func() {
        scanner := bufio.NewScanner(strings.NewReader(urls))
        for scanner.Scan() {
            fmt.Printf("Start URL [%s]\n", scanner.Text())
            out <- scanner.Text()
        }
        close(out)
    }()
    
    return out
}

func parseContent(in <-chan string) (out chan Result) {
    out = make(chan Result)
    go func() {
        for url := range in {
            fastTransport := &http.Transport{DisableKeepAlives : true, ResponseHeaderTimeout : 20 * time.Second}
            fastClient := &http.Client{Transport : fastTransport}
            resp, err := fastClient.Get(url)
            if err != nil {
                fmt.Printf("Error getting content by URL [%s]\n", url)
                continue
            }

            if body, err := ioutil.ReadAll(resp.Body); err == nil {
                out <- countSubstrings(url, body)
            } else {
                fmt.Printf("Error loading Body from URL [%s]\n", url)
            }

            resp.Body.Close()
        }
        close(out)
    }()

    return out
}

/*
 
 */

func merge(grabbers []<-chan Result) (out chan Result) {
    out = make(chan Result)
    var wg sync.WaitGroup

    outer := func(grabber <-chan Result) {
        for grab := range grabber {
            out <- grab
        }
        wg.Done()
    }
    
    wg.Add(len(grabbers))
    for _, grabber := range grabbers {
        go outer(grabber)
    }
    
    go func() {
        wg.Wait()
        close(out)
    }()

    return out
}

func countSubstrings(url string, content []byte) (out Result) {
    body := string(content)
    return Result {
        Url   : url,
        Count : strings.Count(body, TheWord)}
}