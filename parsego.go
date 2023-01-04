package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

// FuncDef describes a single function and it's arguments
type FuncDef struct {
	Name      string    // Name of function or method (e.g. "Get" or "GetN")
	Arguments []argType // List of expected arguments
}

// A Package defines how the parser should be able to locate translation functions
type Package struct {
	Prefix    []string  // List of package- or struct-paths that the functions are attached to
	Functions []FuncDef // List of functions that handle translations
}

// Default package list
var pkgList = []Package{
	{
		Prefix: []string{"github.com/leonelquinteros/gotext.",
			"(*github.com/leonelquinteros/gotext.Locale)",
			"(*github.com/leonelquinteros/gotext.Mo)",
			"(*github.com/leonelquinteros/gotext.Po)"},
		Functions: []FuncDef{
			{Name: "Get", Arguments: []argType{argTypeSingular}},
			{Name: "GetN", Arguments: []argType{argTypeSingular, argTypePlural}},
			{Name: "GetD", Arguments: []argType{argTypeDomain, argTypeSingular}},
			{Name: "GetND", Arguments: []argType{argTypeDomain, argTypeSingular, argTypePlural}},
			{Name: "GetC", Arguments: []argType{argTypeSingular, argTypeContext}},
			{Name: "GetNC", Arguments: []argType{argTypeSingular, argTypePlural, argTypeSkip, argTypeContext}},
			{Name: "GetDC", Arguments: []argType{argTypeDomain, argTypeSingular, argTypeContext}},
			{Name: "GetNDC", Arguments: []argType{argTypeDomain, argTypeSingular, argTypePlural, argTypeSkip, argTypeContext}},
		},
	},
}

type visitor struct {
	basePath  string // Directory we started from
	msgHolder *MsgHolder
	pkg       *packages.Package
}

type argType int

const (
	argTypeSingular argType = iota
	argTypePlural
	argTypeContext
	argTypeDomain
	argTypeSkip
)

func ArgTypeFromString(s string) argType {
	switch strings.ToLower(s) {
	case "singular", "single":
		return argTypeSingular
	case "plural":
		return argTypePlural
	case "context", "ctx":
		return argTypeContext
	case "domain", "dom":
		return argTypeDomain
	case "skip":
		return argTypeSkip
	default:
		fmt.Fprintf(os.Stderr, "unknown argument type %s, defaulting to 'skip'\n", s)
		return argTypeSkip
	}
}

// entryParses parses a single GetX-call
type entryParser struct {
	basePath string // Directory we started from

	// gotext argument types (singular/plural/context/domain)
	argumentTypes []argType
	msgHolder     *MsgHolder

	// Filled when run
	parsedFuncName bool
	currentArg     int
	mismatch       bool

	position string
	singular string
	plural   string
	context  string
	domain   string
}

func (e *entryParser) Visit(node ast.Node) ast.Visitor {
	// First argument is the function name
	if !e.parsedFuncName {
		e.parsedFuncName = true
		return nil
	}

	currentArg := e.currentArg
	e.currentArg++

	// We're not interested in the other arguments
	if currentArg > len(e.argumentTypes)-1 || e.mismatch {
		return nil
	}

	// Skip this argument
	if e.argumentTypes[currentArg] == argTypeSkip {
		return nil
	}

	basic, ok := node.(*ast.BasicLit)
	if !ok || basic.Kind != token.STRING {
		e.mismatch = true
		return nil
	}

	strVal := basic.Value
	quote := strVal[0:1]
	strVal = strings.TrimSuffix(strings.TrimPrefix(strVal, quote), quote)

	switch e.argumentTypes[currentArg] {
	case argTypeSingular:
		e.singular = strVal
	case argTypePlural:
		e.plural = strVal
	case argTypeDomain:
		e.domain = strVal
	case argTypeContext:
		e.context = strVal
	}

	if currentArg == len(e.argumentTypes)-1 {

		e.msgHolder.Add(TranslationString{
			Position: strings.TrimPrefix(strings.TrimPrefix(e.position, e.basePath), "/"),
			Singular: e.singular,
			Plural:   e.plural,
			Context:  e.context,
			Domain:   e.domain,
		})
		return nil
	}

	return nil
}

// selectorAndFunc tries to get the selector and function from call expression.
// For example, given the call expression representing "a.b()", the selector
// is "a.b" and the function is "b" itself.
//
// The final return value will be true if it is able to do extract a selector
// from the call and look up the function object it refers to.
//
// If the call does not include a selector (like if it is a plain "f()" function call)
// then the final return value will be false.
func (v *visitor) selectorAndFunc(call *ast.CallExpr) (*ast.SelectorExpr, *types.Func, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, nil, false
	}

	fn, ok := v.pkg.TypesInfo.ObjectOf(sel.Sel).(*types.Func)
	if !ok {
		// Shouldn't happen, but be paranoid
		return nil, nil, false
	}

	return sel, fn, true

}

func (v *visitor) visitTransFn(pkg Package, call *ast.CallExpr, sel *ast.SelectorExpr) ast.Visitor {
	var argumentTypes []argType

	for _, fn := range pkg.Functions {
		if sel.Sel.Name == fn.Name {
			argumentTypes = fn.Arguments
			break
		}
	}

	if argumentTypes == nil {
		return v
	}

	pos := v.pkg.Fset.Position(call.Lparen)
	return &entryParser{
		basePath:      v.basePath,
		argumentTypes: argumentTypes,
		msgHolder:     v.msgHolder,
		position:      fmt.Sprintf("%s:%d", pos.Filename, pos.Line),
	}
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return v
	}

	sel, fn, ok := v.selectorAndFunc(call)
	if !ok {
		return v
	}

	fullName := fn.FullName()

	for _, pkg := range pkgList {
		for _, prefix := range pkg.Prefix {
			if strings.HasPrefix(fullName, prefix) {
				return v.visitTransFn(pkg, call, sel)
			}
		}
	}

	return v
}

func parseGo(basePath string, folderList []string, msgHolder *MsgHolder) error {
	// Remember current file to write comments on .po file

	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  basePath,
	}

	packages, err := packages.Load(cfg, folderList...)
	if err != nil {
		return err
	}

	for _, pkg := range packages {
		v := &visitor{
			basePath:  basePath,
			msgHolder: msgHolder,
			pkg:       pkg,
		}

		for _, astFile := range v.pkg.Syntax {
			ast.Walk(v, astFile)
		}
	}
	return nil
}
