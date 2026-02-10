// Package service provides helper utilities for the stock gRPC service implementation.
package service

import (
	E "github.com/IBM/fp-go/either"
	IOE "github.com/IBM/fp-go/ioeither"
)

// toEither converts lazy IOEither to eager Either evaluation.
func toEither[ERR, A any](ioe IOE.IOEither[ERR, A]) E.Either[ERR, A] {
	return ioe()
}
