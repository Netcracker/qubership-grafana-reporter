// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type ctxKey string

const (
	ContextMain        = "mainCtx"
	ContextSd          = "shutdownCtx"
	ContextKey  ctxKey = "ctxKey"
)

type ReleaseFunc func(context.Context)

func Shutdown(ctx context.Context, timeout time.Duration, finalizers ...ReleaseFunc) error {
	slog.Default().Info(fmt.Sprintf("Trying to shut down gracefully, timeout %s", timeout.String()))
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	done := make(chan struct{})
	go func(ctx context.Context) {
		for _, release := range finalizers {
			release(ctx)
		}
		done <- struct{}{}
	}(context.WithValue(ctx, ContextKey, ContextSd))
	select {
	case <-done:
		slog.Default().Info("Resources have been released")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("failed to release resources in time")
	}
}
