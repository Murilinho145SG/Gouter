package timerLimiter

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Murilinho145SG/router"
)

type Timeout struct {
	Attempts uint
	Time     time.Time
}

var (
	timerLimiter      = make(map[string]*Timeout)
	maxAttempts  uint = 10
)

func SetMaxAttempts(attempts uint) {
	maxAttempts = attempts
}

func TimerLimiter(handler func(ctx *router.Context)) func(ctx *router.Context) {
	return func(ctx *router.Context) {
		ip := ctx.R.RemoteAddr

		if timerLimiter[ip] != nil {
			tUser := timerLimiter[ip]
			log.Println(tUser.Attempts, maxAttempts)
			if tUser.Attempts == maxAttempts && time.Since(tUser.Time) < 5*time.Minute {
				ctx.WriteError(http.StatusTooManyRequests, errors.New("too many requests"))
				tUser.Time = time.Now()
				return
			}

			if tUser.Attempts == maxAttempts && time.Since(tUser.Time) > 5*time.Minute {
				delete(timerLimiter, ip)
			}
		}

		handler(ctx)
	}
}

func AddAttempt(ctx *router.Context) {
	ip := ctx.R.RemoteAddr

	if timerLimiter[ip] == nil {
		timerLimiter[ip] = &Timeout{Attempts: 1, Time: time.Now()}
	} else {
		timerLimiter[ip].Attempts++
		timerLimiter[ip].Time = time.Now()
	}
}

func RemoveAttempt(ctx *router.Context) {
	ip := ctx.R.RemoteAddr

	if timerLimiter[ip] != nil {
		delete(timerLimiter, ip)
	}
}
