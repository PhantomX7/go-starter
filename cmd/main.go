package main

import (
	_ "ariga.io/atlas-provider-gorm/gormschema"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.Run(":7000")
}
