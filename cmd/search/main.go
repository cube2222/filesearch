package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/blevesearch/bleve"
	"github.com/ryanuber/columnize"
)

var indexPath = "C:/tmp/index.bleve"

func main() {
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
	defer index.Close()

	q := bleve.NewFuzzyQuery(os.Args[1])
	q.SetFuzziness(2)
	r := bleve.NewSearchRequest(q)
	r.Fields = []string{"name", "path", "content"}
	res, err := index.Search(r)
	if err != nil {
		log.Fatal(err)
	}
	// może fuzzy suggester poprawia a potem na tym prefix

	q2 := bleve.NewPrefixQuery(os.Args[1])
	r2 := bleve.NewSearchRequest(q2)
	r2.Fields = []string{"name", "path", "content"}
	res2, err := index.Search(r2)
	if err != nil {
		log.Fatal(err)
	}

	outLines := []string{"Name\tPath"}

	for _, hit := range res.Hits {
		outLines = append(outLines, fmt.Sprintf("%v\t%v", path.Base(hit.Fields["path"].(string)), hit.Fields["path"]))
	}
	for _, hit := range res2.Hits {
		outLines = append(outLines, fmt.Sprintf("%v\t%v", path.Base(hit.Fields["path"].(string)), hit.Fields["path"]))
	}

	conf := columnize.DefaultConfig()
	conf.Delim = "\t"
	fmt.Print(columnize.Format(outLines, conf))
}
