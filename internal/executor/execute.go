package executor

import (
	"fmt"
	"go/ast"
	"plugin"

	"github.com/denizgursoy/cacik/pkg/models"
)

func ExcuteFunction(fnDec *ast.FuncDecl) {

	plug, err := plugin.Open("/home/dgursoy/projects/go/src/cacik-test/test.so")
	if err != nil {
		return
	}

	lookup, err := plug.Lookup("Method1")
	if err != nil {
		return
	}

	f, ok := lookup.(func() models.Config)
	if !ok {
		return
	}
	config := f()
	fmt.Println(config)
}
