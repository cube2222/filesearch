package main

import (
	"context"
	"log"
	"os"

	"github.com/cube2222/search/indexer/indexer"
)

var indexPath = "C:/tmp/index.bleve"

func main() {
	ctx := context.Background()

	index, err := indexer.NewIndexer(ctx, indexPath)
	if err != nil {
		log.Fatal(err)
	}
	err = index.WalkDir(ctx, os.Args[1], true)
	if err != nil {
		log.Fatal(err)
	}

	index.Close()
}
