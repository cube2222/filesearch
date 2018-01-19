package extractors

import (
	"io/ioutil"
	"os"

	"github.com/cube2222/search"
	"github.com/pkg/errors"
)

type textExtractor struct {
}

func NewTextExtractor() search.ContentExtractor {
	return &textExtractor{}
}

func (*textExtractor) FileTypes() []string {
	return []string{".txt"}
}

func (*textExtractor) Extract(file *search.File) (string, error) {
	f, err := os.Stat(file.Path)
	if err != nil {
		return "", errors.Wrap(err, "Couldn't stat file")
	}
	var data []byte
	if f.Size() < 1024*1024 {
		data, err = ioutil.ReadFile(file.Path)
		if err != nil {
			return "", errors.Wrap(err, "Couldn't read file")
		}
	}

	return string(data), nil
}
