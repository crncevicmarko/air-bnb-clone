package cache

import (
	"fmt"
)

const (
	cacheProducts = "images:%s"
	cacheAll      = "images"
)

func constructKey(id string) string {
	return fmt.Sprintf(cacheProducts, id)
}
