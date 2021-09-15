package document

import "github.com/google/uuid"

type Resource struct {
	GUID uuid.UUID // GUID of the resource
	URI  string    // Internal path of the resource
}
