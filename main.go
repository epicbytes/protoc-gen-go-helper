package main

import (
	"fmt"
	"os"

	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"

	"github.com/epicbytes/protoc-gen-go-helpers/module"
)

func main() {
	var pwd = os.Getenv("PWD")
	moduleFile, err := os.ReadFile(fmt.Sprintf("%s/../../go.mod", pwd))
	if err != nil {

	}
	var path = module.ModulePath(moduleFile)

	pgs.Init(pgs.DebugEnv("HELPER_DEBUG")).
		RegisterModule(module.NewHelper(path)).
		RegisterPostProcessor(pgsgo.GoFmt()).
		Render()
}
