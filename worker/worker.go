package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type worker struct {
	URL    *url.URL
	outDir string
	statCh chan stat
}

type stat struct {
	err       error
	bytesRead int64
	fileName  string
}

func main() {
	linksArg := flag.String("l", "", "list of links divided by space or new line char")
	outDir := flag.String("o", ".", "output directory")
	flag.Parse()

	if *linksArg == "" {
		fmt.Println("Nothing to download")
		os.Exit(0)
	}

	if _, err := os.Stat(*outDir); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var (
		links        = strings.Fields(*linksArg)
		workersStats = make(chan stat)
	)

	for _, l := range links {
		parsedURL, err := url.Parse(l)
		if err != nil {
			fmt.Println(err)
			continue
		}
		work := worker{
			parsedURL, *outDir, workersStats,
		}
		go work.start()
	}

	left := len(links)
	for {
		if left == 0 {
			fmt.Println("All done")
			return
		}
		select {
		case s := <-workersStats:
			if s.err != nil {
				fmt.Println(s.err)
			} else {
				fmt.Printf("\rFinished %v - %v bytes\n", s.fileName, s.bytesRead)
			}
			left--
		default:
			displayProgress()
		}
	}
}

func (w *worker) start() {
	res, err := http.Get(fmt.Sprintf("%s", w.URL))
	if err != nil {
		w.statCh <- stat{
			err: err,
		}
		return
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			fmt.Printf("Error on close: %v", err)
		}
	}()
	name := path.Base(w.URL.Path)
	filePath := filepath.Join(w.outDir, name)
	fmt.Printf("Downloading %v\n", filePath)
	read, err := readToFile(res.Body, filePath)
	if err != nil {
		w.statCh <- stat{
			err: err,
		}
		return
	}
	w.statCh <- stat{
		bytesRead: read,
		fileName:  name,
	}
}

func readToFile(
	reader io.Reader,
	filepath string,
) (int64, error) {
	f, err := os.Create(filepath)
	if err != nil {
		return 0, fmt.Errorf("error creating file: %v\n", err)
	}
	n, err := io.Copy(f, reader)
	if err != nil {
		return 0, fmt.Errorf("error while copying data: %v", err)
	}
	err = f.Close()
	if err != nil {
		return n, fmt.Errorf("error closing the file: %v\n", err)
	}
	return n, nil
}

func displayProgress() {
	load := []string{`-`, `\`, `|`, `/`}
	for _, s := range load {
		fmt.Print(s + "\r")
		time.Sleep(100 * time.Millisecond)
	}
}
