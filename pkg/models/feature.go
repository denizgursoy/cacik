package models

import "context"

type (
	Config struct {
		BeforeAll  func(ctx context.Context) error
		AfterAll   func(ctx context.Context) error
		AfterStep  func(ctx context.Context) error
		BeforeStep func(ctx context.Context) error
	}
)
