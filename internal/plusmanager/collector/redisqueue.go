package collector

import "github.com/router-for-me/CLIProxyAPI/v7/internal/redisqueue"

type RedisQueue struct{}

func (RedisQueue) PopOldest(count int) [][]byte { return redisqueue.PopOldest(count) }
