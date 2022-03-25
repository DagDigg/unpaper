package response

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authenvoy "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	envoy_type_v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/gogo/googleapis/google/rpc"
	"google.golang.org/genproto/googleapis/rpc/status"
)

// OK returns a succesful *CheckResponse with provided headers
func OK(headers []*core.HeaderValueOption) *authenvoy.CheckResponse {
	return &authenvoy.CheckResponse{
		Status: &status.Status{Code: int32(rpc.OK)},
		HttpResponse: &authenvoy.CheckResponse_OkResponse{
			OkResponse: &authenvoy.OkHttpResponse{
				Headers: headers,
			},
		},
	}
}

// KO returns an auth CheckResponse_DeniedResponse.
// It sends a default 403 Forbidden status code
func KO(message string) *authenvoy.CheckResponse {
	return &authenvoy.CheckResponse{
		Status: &status.Status{Code: int32(rpc.PERMISSION_DENIED), Message: message},
		HttpResponse: &authenvoy.CheckResponse_DeniedResponse{
			DeniedResponse: &authenvoy.DeniedHttpResponse{
				Status: &envoy_type_v3.HttpStatus{
					Code: envoy_type_v3.StatusCode_Forbidden,
				},
				Body: message,
			},
		},
	}
}

// GetHeaderValueOptions iterates a [string]string map
// and returns it as []*core.HeaderValueOption
func GetHeaderValueOptions(headers map[string]string) []*core.HeaderValueOption {
	res := []*core.HeaderValueOption{}

	for k, v := range headers {
		res = append(res, &core.HeaderValueOption{Header: &core.HeaderValue{Key: k, Value: v}})
	}

	return res
}
