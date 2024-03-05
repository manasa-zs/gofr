package main

import (
	"net/http"
	"testing"
	"time"

	"gofr.dev/pkg/gofr"
)

//
//func TestIntegration_SimpleAPIServer(t *testing.T) {
//	const host = "http://localhost:9000"
//	go main()
//	time.Sleep(time.Second * 3) // Giving some time to start the server
//
//	tests := []struct {
//		desc       string
//		path       string
//		statusCode int
//	}{
//		{"empty path", "/", 404},
//		{"hello handler", "/hello", 200},
//		{"hello handler with query parameter", "/hello?name=gofr", 200},
//		{"error handler", "/error", 500},
//		{"redis handler", "/redis", 200},
//		{"trace handler", "/trace", 200},
//		{"mysql handler", "/mysql", 200},
//		{"health handler", "/.well-known/health", 200}, // Health check should be added by the framework.
//		{"favicon handler", "/favicon.ico", 200},       //Favicon should be added by the framework.
//	}
//
//	for i, tc := range tests {
//		req, _ := http.NewRequest("GET", host+tc.path, nil)
//		c := http.Client{}
//		resp, err := c.Do(req)
//
//		assert.Nil(t, err, "TEST[%d], Failed.\n%s", i, tc.desc)
//
//		assert.Equal(t, tc.statusCode, resp.StatusCode, "TEST[%d], Failed.\n%s", i, tc.desc)
//	}
//}
//
//func TestRedisHandler(t *testing.T) {
//	a := gofr.New()
//	logger := logging.NewLogger(logging.DEBUG)
//	redisClient, mock := redismock.NewClientMock()
//
//	rc := redis.NewClient(testutil.NewMockConfig(map[string]string{"REDIS_HOST": "localhost", "REDIS_PORT": "2001"}), logger, a.Metrics())
//	rc.Client = redisClient
//
//	mock.ExpectGet("test").SetErr(testutil.CustomError{ErrorMessage: "redis get error"})
//
//	ctx := &gofr.Context{Context: context.Background(),
//		Request: nil, Container: &container.Container{Logger: logger, Redis: rc}}
//
//	resp, err := RedisHandler(ctx)
//
//	assert.Nil(t, resp)
//	assert.NotNil(t, err)
//}

//func BenchmarkGoFrLoggingWithoutServer(b *testing.B) {
//	l := logging.NewLogger(logging.INFO)
//
//	for n := 0; n < b.N; n++ {
//		for i := 0; i < 100000; i++ {
//			l.Info("Benchmarking log performance")
//		}
//	}
//}

func BenchmarkGoFrLoggingHTTPServer(b *testing.B) {
	a := gofr.New()

	a.GET("/benchmark", func(c *gofr.Context) (interface{}, error) {
		c.Log("Processing benchmark request")
		return "Benchmark completed", nil
	})

	go func() {
		a.Run()
	}()

	time.Sleep(2 * time.Second)

	done := make(chan struct{})

	// Send requests concurrently in a goroutine
	go func() {
		for i := 0; i < 10000; i++ {
			resp, err := http.Get("http://localhost:9000/benchmark")
			if err != nil {
				b.Fatalf("failed to send request: %v", err)
			}

			resp.Body.Close()
		}
		done <- struct{}{}
	}()

	<-done
}
