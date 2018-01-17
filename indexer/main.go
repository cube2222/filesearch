package main

import (
	"context"
	"io/ioutil"
	"path"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"os"
	"strings"
	"github.com/blevesearch/bleve"
	"sync"
)

var indexPath = "C:/tmp/index.bleve"

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	indexer := Indexer{}
	err := indexer.Init(ctx)
	if err != nil {
		log.Fatal(err)
	}
	err = indexer.WalkDir(ctx, os.Args[1], true)
	if err != nil {
		log.Fatal(err)
	}

	cancel()
	indexer.Close()
}

type Indexer struct {
	index      bleve.Index
	docChannel chan *File
	wg         sync.WaitGroup
}

func (indexer *Indexer) Init(ctx context.Context) error {
	var index bleve.Index

	if _, err := os.Stat(indexPath); err != nil {
		index, err = bleve.New(indexPath, bleve.NewIndexMapping())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		index, err = bleve.Open(indexPath)
		if err != nil {
			log.Fatal(err)
		}
	}

	indexer.index = index
	indexer.docChannel = make(chan *File, 8000)

	indexer.wg.Add(1)
	go Commiter(ctx, &indexer.wg, index, indexer.docChannel)

	return nil
}

func (indexer *Indexer) Close() {
	indexer.wg.Wait()
	indexer.index.Close()
}

type File struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

func (indexer *Indexer) WalkDir(ctx context.Context, dir string, toplevel bool) error {
	dirInfo, err := ioutil.ReadDir(dir)
	if err != nil {
		return errors.Wrapf(err, "Couldn't read directory", dir)
	}

	for _, fileInfo := range dirInfo {
		if fileInfo.IsDir() {
			if !strings.Contains(strings.ToLower(fileInfo.Name()), "windows") {
				dirPath := path.Join(dir, fileInfo.Name())
				if toplevel {
					log.Println(dirPath)
				}
				err := indexer.WalkDir(ctx, dirPath, false)
				if err != nil {
					fmt.Printf("Couldn't walk directory: %v \n Ignoring \n Error: %v\n", dirPath, err)
					continue
				}
			} else {
				fmt.Printf("Omitting windows: %v\n", path.Join(dir, fileInfo.Name()))
			}
		}
		file := File{
			Name: fileInfo.Name(),
			Path: path.Join(dir, fileInfo.Name()),
		}
		if i := strings.LastIndex(file.Name, "."); i != -1 {
			file.Name = file.Name[:i]
		}

		indexer.docChannel <- &file
	}

	return nil
}

func Commiter(ctx context.Context, wg *sync.WaitGroup, index bleve.Index, files <-chan *File) {
	batch := index.NewBatch()
	for {
		for i := 0; i < 2048; i++ {
			select {
			case file := <-files:
				err := batch.Index(file.Path, file)
				if err != nil {
					fmt.Println(err)
					continue
				}
			case <-ctx.Done():
				err := index.Batch(batch)
				if err != nil {
					fmt.Println(err)
				}
				batch.Reset()
				wg.Done()
				return
			}
		}
		err := index.Batch(batch)
		if err != nil {
			fmt.Println(err)
		}
		batch.Reset()
	}
}
