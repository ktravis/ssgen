package main // import "github.com/ktravis/ssgen"

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

var (
	in             = flag.String("in", "src", "`dir` for source markdown")
	out            = flag.String("out", "build", "`dir` for output")
	templates      = flag.String("templates", "templates", "`dir` for input templates")
	serve          = flag.String("serve", "", "watch files, build and serve output at `address`")
	static         = flag.String("static", "static", "`dir` containing static files to be served at '/static'")
	debug          = flag.Bool("debug", false, "print debug messages")
	enableReloader = flag.Bool("reloader", false, "include reloader js code with -serve")
)

func dbg(s string, args ...any) {
	if *debug {
		log.Printf("[debug] "+s, args...)
	}
}

func init() {
	flag.Parse()
}

func main() {
	log.Printf("compiling '%s' to '%s'...", *in, *out)
	s, err := loadContent()
	if err != nil {
		log.Fatal(err)
	}

	if err := s.compile(); err != nil {
		log.Fatal(err)
	}

	reloadChan := make(chan bool)

	if *serve != "" {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}

		defer watcher.Close()
		watcher.Add(*in)
		watcher.Add(filepath.Join(*templates, "include"))
		watcher.Add(*templates)
		err = filepath.Walk(*in, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			if info.IsDir() {
				watcher.Add(path)
			}
			return nil
		})
		go func() {
			for {
				select {
				case event := <-watcher.Events:
					if event.Op&fsnotify.Write != 0 {
						dbg("watched file (%v) changed", event)
						log.Printf("compiling '%s' to '%s'...", *in, *out)
						s, err := loadContent()
						if err != nil {
							log.Printf("compilation error: %v", err)
							continue
						}

						if err := s.compile(); err != nil {
							log.Printf("compilation error: %v", err)
						}
						select {
						case reloadChan <- true:
						default:
						}
					}
				case err := <-watcher.Errors:
					log.Printf("watcher error: %v", err)
				}
			}
		}()

		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(*static))))
		if *enableReloader {
			http.Handle("/reload", reloader{reloadChan})
		}
		http.Handle("/", FileServer{http.Dir(*out)})
		log.Printf("watching files and serving at '%s'", *serve)
		log.Fatal(http.ListenAndServe(*serve, nil))
	}
}
