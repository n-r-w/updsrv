package entity

import "context"

// ClientInfo информация о клиенте
type ClientInfo struct {
	IP string
}

type clientInfoKeyType string

const clientInfoKey = clientInfoKeyType("ClientInfoKey")

func PutClientInfoToContext(c *ClientInfo, ctx context.Context) context.Context {
	return context.WithValue(ctx, clientInfoKey, c)
}

func GetClientInfoFromContext(ctx context.Context) *ClientInfo {
	if c, ok := ctx.Value(clientInfoKey).(*ClientInfo); ok {
		return c
	}
	return nil
}
