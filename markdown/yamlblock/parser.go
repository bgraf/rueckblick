package yamlblock

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v2"
)

var yamlBlockParserNodeKind = ast.NewNodeKind("yamlBlockParserNode")

type yamlBlockParserNode struct {
	ast.BaseBlock
	error         error
	addin         Addin
	object        interface{}
	immediateStop bool
}

func (n *yamlBlockParserNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

func (n *yamlBlockParserNode) Kind() ast.NodeKind {
	return yamlBlockParserNodeKind
}

type yamlBlockParser struct {
	parent *yamlBlock
}

func newYamlBlockParser(parent *yamlBlock) parser.BlockParser {
	return &yamlBlockParser{
		parent: parent,
	}
}

func (b *yamlBlockParser) Trigger() []byte {
	return []byte{':'}
}

func (b *yamlBlockParser) Open(node ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()

	fields := strings.Fields(string(line))

	// Syntax is "::" name ["---"]
	// If the trailing "---" is not provided, then all default values are assumed

	if len(fields) < 2 || fields[0] != "::" || (len(fields) >= 3 && fields[2] != "---") {
		return nil, parser.NoChildren
	}

	key := strings.ToLower(fields[1])

	addin := b.parent.findAddin(key)
	if addin == nil {
		return nil, parser.NoChildren
	}

	reader.Advance(segment.Len() - 1)

	n := &yamlBlockParserNode{
		addin: addin,
	}

	if len(fields) == 2 {
		n.immediateStop = true
	}

	return n, parser.NoChildren
}

func (b *yamlBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	if node.(*yamlBlockParserNode).immediateStop {
		return parser.Close
	}

	line, seg := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	if lineStr == "---" {
		reader.Advance(seg.Len())
		return parser.Close
	}

	node.Lines().Append(seg)

	return parser.Continue | parser.NoChildren
}

func (b *yamlBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	lines := node.Lines()
	var buf bytes.Buffer
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		buf.Write(segment.Value(reader.Source()))
	}

	storeYamlObject(node, pc, buf.Bytes())
}

func storeYamlObject(node ast.Node, pc parser.Context, raw []byte) {
	ynode := node.(*yamlBlockParserNode)
	obj := ynode.addin.Make(pc)

	var err error

	if raw != nil {
		err = yaml.Unmarshal(raw, obj)
	}

	if err != nil {
		ynode.error = err
	} else {
		ynode.object = obj
	}
}

func (b *yamlBlockParser) CanInterruptParagraph() bool {
	return false
}

func (b *yamlBlockParser) CanAcceptIndentedLine() bool {
	return false
}
