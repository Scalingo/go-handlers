package handlers

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Scalingo/go-utils/logger"
)

const (
	username = "username"
	password = "password"
)

func TestProfilingRouterActivation(t *testing.T) {
	t.Run("does nots start profiling if pprofEnabled not set", func(t *testing.T) {
		// arrange
		t.Setenv("PPROF_ENABLED", "")
		ctx := context.Background()

		// act
		profilingRouter, err := NewProfilingRouter(ctx)

		// assert
		require.NoError(t, err)
		assert.False(t, isProfilingEnabled(profilingRouter))
	})

	t.Run("does nots start profiling if not a bool", func(t *testing.T) {
		// arrange
		t.Setenv("PPROF_ENABLED", "obviouslyNotABool")
		ctx := context.Background()

		// act
		profilingRouter, err := NewProfilingRouter(ctx)

		// assert
		require.Error(t, err, "fail to initialize pprof profiling: fail to parse environment variable PPROF_ENABLED: strconv.ParseBool: parsing \"obviouslyNotABool\": invalid syntax")
		assert.Nil(t, profilingRouter)
	})

	t.Run("does nots start profiling if not enabled", func(t *testing.T) {
		// arrange
		t.Setenv("PPROF_ENABLED", "false")
		ctx := context.Background()

		// act
		profilingRouter, err := NewProfilingRouter(ctx)

		// assert
		require.NoError(t, err)
		assert.False(t, isProfilingEnabled(profilingRouter))
	})

	t.Run("does nots start profiling if username is not set", func(t *testing.T) {
		// arrange
		t.Setenv("PPROF_ENABLED", "true")
		t.Setenv("PPROF_PASSWORD", password)
		ctx := context.Background()

		// act
		profilingRouter, err := NewProfilingRouter(ctx)

		// assert
		require.NoError(t, err)
		assert.False(t, isProfilingEnabled(profilingRouter))
	})

	t.Run("does nots start profiling if password is not set", func(t *testing.T) {
		// arrange
		t.Setenv("PPROF_ENABLED", "true")
		t.Setenv("PPROF_USERNAME", username)
		ctx := context.Background()

		// act
		profilingRouter, err := NewProfilingRouter(ctx)

		// assert
		require.NoError(t, err)
		assert.False(t, isProfilingEnabled(profilingRouter))
	})

	t.Run("starts profiling if all environment variables are set", func(t *testing.T) {
		// arrange
		t.Setenv("PPROF_ENABLED", "true")
		t.Setenv("PPROF_USERNAME", username)
		t.Setenv("PPROF_PASSWORD", password)
		ctx := context.Background()

		// act
		profilingRouter, err := NewProfilingRouter(ctx)

		// assert
		require.NoError(t, err)
		assert.True(t, isProfilingEnabled(profilingRouter))
	})
}

func TestProfilingRouterNotExistingRoute(t *testing.T) {
	t.Run("responds not found", func(t *testing.T) {
		// arrange
		path := PprofRoutePrefix + "/path_does_not_exist"

		t.Setenv("PPROF_ENABLED", "true")
		t.Setenv("PPROF_USERNAME", username)
		t.Setenv("PPROF_PASSWORD", password)

		ctx := createLog()

		profilingRouter, err := NewProfilingRouter(ctx)
		require.NoError(t, err)

		httpRecorder := httptest.NewRecorder()

		request := createGetRequest(t, path)

		// act
		profilingRouter.ServeHTTP(httpRecorder, request)

		// assert
		assert.Equal(t, http.StatusNotFound, httpRecorder.Code)
	})
}

func TestProfilingRouterEndpointWithoutAuth(t *testing.T) {
	t.Run("returns unauthorized", func(t *testing.T) {
		// arrange
		t.Setenv("PPROF_ENABLED", "true")
		t.Setenv("PPROF_USERNAME", username)
		t.Setenv("PPROF_PASSWORD", password)

		ctx := createLog()

		profilingRouter, err := NewProfilingRouter(ctx)
		require.NoError(t, err)

		httpRecorder := httptest.NewRecorder()

		request := createGetRequest(t, PprofRoutePrefix+"/")

		// act
		profilingRouter.ServeHTTP(httpRecorder, request)

		// assert
		assert.Equal(t, http.StatusUnauthorized, httpRecorder.Code)
	})
}

func TestProfilingRouterEndpoint(t *testing.T) {
	pathsToTest := []string{
		"/",
		"/profile?seconds=1",
		"/symbol",
		"/cmdline",
		"/trace",
		"/allocs",
		"/heap",
		"/goroutine",
		"/mutex",
		"/block",
		"/threadcreate",
	}

	for _, path := range pathsToTest {
		t.Run("returns pprof["+path+"]", func(t *testing.T) {
			// arrange
			t.Setenv("PPROF_ENABLED", "true")
			t.Setenv("PPROF_USERNAME", username)
			t.Setenv("PPROF_PASSWORD", password)

			ctx := createLog()

			profilingRouter, err := NewProfilingRouter(ctx)
			require.NoError(t, err)

			httpRecorder := httptest.NewRecorder()

			request := createGetRequest(t, PprofRoutePrefix+path)
			request = addAuthorization(request)

			// act
			profilingRouter.ServeHTTP(httpRecorder, request)

			// assert
			assert.Equal(t, http.StatusOK, httpRecorder.Code)
		})
	}
}

// isProfilingEnabled checks if the pprof prefix route is registered
func isProfilingEnabled(profilingRouter *Router) bool {
	req := httptest.NewRequest(http.MethodGet, PprofRoutePrefix, nil)

	m := &mux.RouteMatch{}
	return profilingRouter.Match(req, m)
}

func createLog() context.Context {
	log := logger.Default()
	return logger.ToCtx(context.Background(), log)
}

func createGetRequest(t *testing.T, path string) *http.Request {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}

	return req
}

func addAuthorization(request *http.Request) *http.Request {
	auth := username + ":" + password
	request.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))

	return request
}
