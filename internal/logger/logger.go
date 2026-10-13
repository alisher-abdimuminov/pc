package logger

import "go.uber.org/zap"

func New(prod bool) (*zap.Logger, error) {
	if prod {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
