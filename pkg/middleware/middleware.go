package middleware

import (
	"github.com/arthurlch/goryu/pkg/context"
)

type Middleware func(context.HandlerFunc) context.HandlerFunc