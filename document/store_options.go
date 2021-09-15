package document

type StoreOptions struct {
	MapGPXResource   func(doc *Document, srcPath string) (Resource, bool)
	MapImageResource func(doc *Document, galleryNo int, srcPath string) (Resource, bool)
}
