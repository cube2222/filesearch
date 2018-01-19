package indexer

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/blevesearch/bleve"
	"github.com/cube2222/search"
	"github.com/cube2222/search/indexer/extractors"
	"github.com/pkg/errors"
)

type Indexer struct {
	extractorWg      sync.WaitGroup
	extractorChannel chan *search.File

	index         bleve.Index
	commitWg      sync.WaitGroup
	commitChannel chan *search.File

	extractors map[string]search.ContentExtractor
}

func NewIndexer(ctx context.Context, indexPath string) (*Indexer, error) {
	indexer := Indexer{}

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

	indexer.extractorChannel = make(chan *search.File, 8000)
	indexer.commitChannel = make(chan *search.File, 8000)

	indexer.extractors = make(map[string]search.ContentExtractor)
	textExtractor := extractors.NewTextExtractor()
	for _, ext := range textExtractor.FileTypes() {
		indexer.extractors[ext] = textExtractor
	}

	for i := 0; i < 8; i++ {
		indexer.extractorWg.Add(1)
		go indexer.Extractor(&indexer.extractorWg, indexer.extractorChannel, indexer.commitChannel)
	}

	indexer.index = index
	indexer.commitWg.Add(1)
	go indexer.Commiter(&indexer.commitWg, indexer.commitChannel)

	return &indexer, nil
}

func (indexer *Indexer) Close() {
	close(indexer.extractorChannel)

	indexer.extractorWg.Wait()
	close(indexer.commitChannel)

	indexer.commitWg.Wait()
	err := indexer.index.Close()
	if err != nil {
		log.Fatal(err)
	}
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
		file := search.File{
			Name: fileInfo.Name(),
			Path: path.Join(dir, fileInfo.Name()),
		}
		if i := strings.LastIndex(file.Name, "."); i != -1 {
			file.Name = file.Name[:i]
		}

		indexer.extractorChannel <- &file
	}

	return nil
}

func (indexer *Indexer) Commiter(wg *sync.WaitGroup, files <-chan *search.File) {
	batch := indexer.index.NewBatch()
	i := 0
	for file := range files {
		err := batch.Index(file.Path, file)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if i >= 1024 {
			err := indexer.index.Batch(batch)
			if err != nil {
				fmt.Println(err)
			}
			batch.Reset()
			i = 0
		}
		i++
	}
	err := indexer.index.Batch(batch)
	if err != nil {
		fmt.Println(err)
	}
	batch.Reset()

	wg.Done()
}

func (indexer *Indexer) Extractor(wg *sync.WaitGroup, in chan *search.File, out chan<- *search.File) {
	for file := range in {
		extractor, ok := indexer.extractors[path.Ext(file.Path)]
		if ok {
			content, err := extractor.Extract(file)
			if err != nil {
				log.Printf("Couldn't extract content from file: %v Error: %v\n", file.Path, err)
				continue
			}
			file.Content = content
		}
		out <- file
	}

	wg.Done()
}
