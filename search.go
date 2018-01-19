package search


// Docx regexp: \<[a-zA-Z0-9ąćęłóśźż:/=_,#\.\;\"\ \-\(\)?@]*\>
// ale znaki jak ść powinny znikać stripperem unicode'u
type ContentExtractor interface {
	FileTypes() []string
	Extract(file *File) (string, error)
}

type File struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Content string `json:"content"`
}
