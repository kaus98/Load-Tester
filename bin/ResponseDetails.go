package bin

import (
	"time"
)

type ResponseDetails struct {
	StartTime time.Time
	EndTime   time.Time
	Code      int
}
