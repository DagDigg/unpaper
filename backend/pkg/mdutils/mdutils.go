package mdutils

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/metadata"
)

// GetFirstServerMDValue returns the first server header metadata value for the provided key if found
func GetFirstServerMDValue(md runtime.ServerMetadata, key string) (string, bool) {
	values := md.HeaderMD.Get(key)
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

// GetFirstMDValue returns the first metadata value for the provided key if found
func GetFirstMDValue(md metadata.MD, key string) (string, bool) {
	return getFirstMDValue(md, key)
}

func getFirstMDValue(md metadata.MD, key string) (string, bool) {
	values := md.Get(key)
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

// GetUserIDFromMD returns the 'x-user-id' from metadata
func GetUserIDFromMD(ctx context.Context) (string, bool) {
	md, ok := getMetadataFromCtx(ctx)
	if !ok {
		return "", false
	}
	userID, ok := getFirstMDValue(md, "x-user-id")
	return userID, ok
}

func getMetadataFromCtx(ctx context.Context) (metadata.MD, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, false
	}
	return md, true
}
