package serve

import (
	"path/filepath"

	"github.com/google/uuid"
)

type resourceMap struct {
	bySrc map[string]uuid.UUID
	byDst map[uuid.UUID]string
}

func newResourceMap() *resourceMap {
	return &resourceMap{
		bySrc: make(map[string]uuid.UUID),
		byDst: make(map[uuid.UUID]string),
	}
}

func (r *resourceMap) IDFromPath(srcPath string) uuid.UUID {
	var err error
	srcPath, err = filepath.Abs(srcPath)
	if err != nil {
		panic(err)
	}

	if dstName, ok := r.bySrc[srcPath]; ok {
		return dstName
	}

	guid, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}

	r.bySrc[srcPath] = guid
	r.byDst[guid] = srcPath

	return guid
}

func (r *resourceMap) PathFromID(guid uuid.UUID) (string, bool) {
	if src, ok := r.byDst[guid]; ok {
		return src, true
	}

	return "", false
}
