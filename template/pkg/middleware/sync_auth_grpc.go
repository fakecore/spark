package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type SyncAuthConfig struct {
	Enabled         bool
	WorkspaceTokens map[string]string
	DefaultToken    string

	// Optional fine-grained tokens (capabilities).
	WorkspaceReadTokens  map[string]string
	WorkspaceWriteTokens map[string]string
	DefaultReadToken     string
	DefaultWriteToken    string
}

// SyncAuthUnaryServerInterceptor enforces a workspace-scoped token check for Sync gRPC methods.
//
// Rules:
// - Only intercepts methods with prefix "/api.sync.v1.ChangeSync/".
// - Token sources (first match wins): "x-sync-token", then "authorization: Bearer <token>".
// - WorkspaceID source: request must implement interface{ GetWorkspaceId() string }.
func SyncAuthUnaryServerInterceptor(cfg SyncAuthConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info == nil || !strings.HasPrefix(info.FullMethod, "/api.sync.v1.ChangeSync/") {
			return handler(ctx, req)
		}
		if !cfg.Enabled {
			return handler(ctx, req)
		}

		needWrite := strings.HasSuffix(info.FullMethod, "/PushChanges") ||
			strings.HasSuffix(info.FullMethod, "/RegisterNode") ||
			strings.HasSuffix(info.FullMethod, "/ReportMissingParents") ||
			strings.HasSuffix(info.FullMethod, "/ResolveMissingParent")

		wsGetter, ok := req.(interface{ GetWorkspaceId() string })
		if !ok {
			return nil, status.Error(codes.Internal, "sync auth: request missing workspace_id")
		}
		workspaceID := strings.TrimSpace(wsGetter.GetWorkspaceId())
		if workspaceID == "" {
			return nil, status.Error(codes.InvalidArgument, "workspace_id is required")
		}

		md, _ := metadata.FromIncomingContext(ctx)
		token := firstMD(md, "x-sync-token")
		if token == "" {
			auth := firstMD(md, "authorization")
			auth = strings.TrimSpace(auth)
			auth = strings.TrimPrefix(auth, "Bearer ")
			token = strings.TrimSpace(auth)
		}
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "missing sync token")
		}

		// Workspace-specific tokens take precedence over defaults (prevents default from bypassing a
		// read-only workspace configuration).
		wsFull := ""
		if cfg.WorkspaceTokens != nil {
			wsFull = strings.TrimSpace(cfg.WorkspaceTokens[workspaceID])
		}
		wsRead := ""
		if cfg.WorkspaceReadTokens != nil {
			wsRead = strings.TrimSpace(cfg.WorkspaceReadTokens[workspaceID])
		}
		wsWrite := ""
		if cfg.WorkspaceWriteTokens != nil {
			wsWrite = strings.TrimSpace(cfg.WorkspaceWriteTokens[workspaceID])
		}
		wsConfigured := wsFull != "" || wsRead != "" || wsWrite != ""

		allow := func(full, read, write string) bool {
			if strings.TrimSpace(full) != "" && token == strings.TrimSpace(full) {
				return true
			}
			if needWrite {
				return strings.TrimSpace(write) != "" && token == strings.TrimSpace(write)
			}
			// read: read token OR write token.
			if strings.TrimSpace(read) != "" && token == strings.TrimSpace(read) {
				return true
			}
			return strings.TrimSpace(write) != "" && token == strings.TrimSpace(write)
		}

		if wsConfigured {
			if !allow(wsFull, wsRead, wsWrite) {
				return nil, status.Error(codes.PermissionDenied, "sync token mismatch")
			}
			return handler(ctx, req)
		}

		// Defaults.
		if strings.TrimSpace(cfg.DefaultToken) != "" ||
			strings.TrimSpace(cfg.DefaultReadToken) != "" ||
			strings.TrimSpace(cfg.DefaultWriteToken) != "" {
			if !allow(cfg.DefaultToken, cfg.DefaultReadToken, cfg.DefaultWriteToken) {
				return nil, status.Error(codes.PermissionDenied, "sync token mismatch")
			}
			return handler(ctx, req)
		}

		return nil, status.Error(codes.PermissionDenied, "sync token not configured for workspace")
	}
}

func firstMD(md metadata.MD, key string) string {
	if md == nil {
		return ""
	}
	vs := md.Get(key)
	if len(vs) == 0 {
		// gRPC metadata keys are case-insensitive but normalized; be defensive.
		vs = md.Get(strings.ToLower(key))
	}
	if len(vs) == 0 {
		return ""
	}
	return strings.TrimSpace(vs[0])
}
