package yamlblock

import "github.com/yuin/goldmark/parser"

var yamlblockDocumentPathKey = parser.NewContextKey()

func SetDocumentPath(pc parser.Context, documentPath string) {
	pc.Set(yamlblockDocumentPathKey, documentPath)
}

func DocumentPath(pc parser.Context) (string, bool) {
	if path, ok := pc.Get(yamlblockDocumentPathKey).(string); ok {
		return path, true
	}

	return "", false
}
