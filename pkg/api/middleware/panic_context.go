package middleware

import (
	"context"
	"fmt"
)

type recoveredPanicInfo struct {
	Recovered bool
	Message   string
}

type recoveredPanicInfoKey struct{}

func withRecoveredPanicInfo(ctx context.Context) (context.Context, *recoveredPanicInfo) {
	info := &recoveredPanicInfo{}
	return context.WithValue(ctx, recoveredPanicInfoKey{}, info), info
}

func markRecoveredPanic(ctx context.Context, panicValue interface{}) {
	if ctx == nil {
		return
	}
	info, ok := ctx.Value(recoveredPanicInfoKey{}).(*recoveredPanicInfo)
	if !ok || info == nil {
		return
	}
	info.Recovered = true
	info.Message = fmt.Sprint(panicValue)
}
