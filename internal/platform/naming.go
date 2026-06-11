package platform

import (
	"fmt"
	"time"
)

func ConsumerSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
