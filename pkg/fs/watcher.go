package fs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher *fsnotify.Watcher
	handler func(fsnotify.Event)
	done    chan struct{}
	wg      sync.WaitGroup
}

// NewWatcher creates a new Watcher instance
func NewWatcher(handler func(fsnotify.Event)) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		watcher: w,
		handler: handler,
		done:    make(chan struct{}),
	}

	watcher.wg.Add(1)
	go watcher.start()

	return watcher, nil
}

// AddFolder recursively adds a folder and its subfolders to the watcher
func (w *Watcher) AddFolder(folder string) error {
	return filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			err = w.watcher.Add(path)
			if err != nil {
				return err
			}
			fmt.Println("Watching:", path)
		}
		return nil
	})
}

// Close shuts down the watcher
func (w *Watcher) Close() {
	close(w.done)
	w.wg.Wait()
	w.watcher.Close()
}

func (w *Watcher) start() {
	defer w.wg.Done()
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handler(event)
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					_ = w.AddFolder(event.Name)
				}
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		case <-w.done:
			return
		}
	}
}
