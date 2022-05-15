package serve

import (
	"net/http"

	"github.com/bgraf/rueckblick/render"
	"github.com/gin-gonic/gin"
)

func (api *serveAPI) ServeIndex(c *gin.Context) {
	groups := render.MakeDocumentGroups(api.store.Documents)
	c.HTML(
		http.StatusOK,
		"index.html",
		gin.H{
			"Groups": groups,
		},
	)
}

func (api *serveAPI) ServeTag(c *gin.Context) {
	tagName := c.Param("tag")

	documents := api.store.DocumentsByTagName(tagName)
	groups := render.MakeDocumentGroups(documents)
	tag, _ := api.store.TagByName(tagName)

	c.HTML(
		http.StatusOK,
		"index.html",
		gin.H{
			"Groups": groups,
			"Tag":    tag.Raw,
		},
	)
}
