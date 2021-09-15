package document

type StoreOptions struct {
	MapGPXResource func(doc *Document, srcPath string) (Resource, bool)
}
