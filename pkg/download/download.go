package download

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sync"
)

// URLs downloads a list of urls and saves the respond in a file.
func URLs(urls []*url.URL, suffix string) {
	var wg sync.WaitGroup
	for _, u := range urls {
		wg.Add(1)
		go func(url *url.URL) {
			defer wg.Done()
			fn := filepath.Clean(path.Base(url.Path)) + "-" + suffix
			out, err := os.Create(fn)
			if err != nil {
				log.Printf("error creating the destination file:%v\n", err)
				return
			}
			defer out.Close()

			resp, err := http.Get(url.String())
			if err != nil {
				log.Printf("error downloading file:%v\n", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("url:%v,bad status: %v\n", url.String(), resp.Status)
				return
			}

			_, err = io.Copy(out, resp.Body)
			if err != nil {
				log.Printf("error while reading the response: %v\n", err)
				return
			}

			log.Printf("downloaded:%v saved as: %v\n", url.String(), fn)

		}(u)
	}
	wg.Wait()
}
