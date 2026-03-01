package http

import (
	"context"
	"time"

	oapi "github.com/bziks/gitlab-package-finder/pkg/oapi"
)

func (api *API) K8sLive(ctx context.Context, request oapi.K8sLiveRequestObject) (oapi.K8sLiveResponseObject, error) {
	return oapi.K8sLive200Response{}, nil
}

func (api *API) K8sReady(ctx context.Context, request oapi.K8sReadyRequestObject) (oapi.K8sReadyResponseObject, error) {
	return oapi.K8sReady200Response{}, nil
}

func (api *API) Ping(_ context.Context, _ oapi.PingRequestObject) (oapi.PingResponseObject, error) {
	return oapi.Ping200JSONResponse{
		Data: "pong",
		Meta: oapi.MetaResponse{
			Timestamp: time.Now(),
		},
	}, nil
}
