package g

import (
	"github.com/dinp/common/model"
	"log"
	"runtime"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

var (
	RealState = model.NewSafeRealState()
)
