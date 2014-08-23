package relax

type MultipartForm struct {
	Fields map[string]string
	Files  map[string]string
}

func NewMultipartForm() *MultipartForm {
	return &MultipartForm{Files: make(map[string]string), Fields: make(map[string]string)}
}
