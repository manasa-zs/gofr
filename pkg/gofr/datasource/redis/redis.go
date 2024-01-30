package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	otel "github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"

	"gofr.dev/pkg/gofr/config"
	"gofr.dev/pkg/gofr/datasource"
)

const (
	redisPingTimeout = 5 * time.Second
	defaultRedisPort = 6379
)

type Config struct {
	HostName string
	Port     int
	Options  *redis.Options
}

type Redis struct {
	*redis.Client
	logger datasource.Logger
	config *Config
}

// NewClient return a redis client if connection is successful based on Config.
// In case of error, it returns an error as second parameter.
func NewClient(c config.Config, logger datasource.Logger) *Redis {
	var redisConfig = &Config{}

	if redisConfig.HostName = c.Get("REDIS_HOST"); redisConfig.HostName == "" {
		return nil
	}

	port, err := strconv.Atoi(c.Get("REDIS_PORT"))
	if err != nil {
		port = defaultRedisPort
	}

	redisConfig.Port = port

	options := new(redis.Options)

	if options.Addr == "" {
		options.Addr = fmt.Sprintf("%s:%d", redisConfig.HostName, redisConfig.Port)
	}

	redisConfig.Options = options

	rc := redis.NewClient(redisConfig.Options)
	rc.AddHook(&redisHook{logger: logger})

	ctx, cancel := context.WithTimeout(context.TODO(), redisPingTimeout)
	defer cancel()

	if err := rc.Ping(ctx).Err(); err != nil {
		logger.Errorf("could not connect to redis at %s:%d. error: %s", redisConfig.HostName, redisConfig.Port, err)

		return &Redis{Client: nil, config: redisConfig, logger: logger}
	}

	if err := otel.InstrumentTracing(rc); err != nil {
		logger.Errorf("could not add tracing instrumentation, error : %s", err)
	}

	logger.Logf("connected to redis at %s:%d", redisConfig.HostName, redisConfig.Port)

	return &Redis{Client: rc, config: redisConfig, logger: logger}
}

// TODO - if we make Redis an interface and expose from container we can avoid c.Redis(c, command) using methods on c and still pass c.
// type Redis interface {
//	Get(string) (string, error)
// }
