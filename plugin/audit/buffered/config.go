package buffered

import "github.com/yubo/golib/api"

// Config represents batching delegate audit backend configuration.
type Config struct {
	// BufferSize defines a size of the buffering queue.
	BufferSize int `json:"bufferSize" description:"The size of the buffer to store events before batching and writing. Only used in batch mode."`
	// MaxBatchSize defines maximum size of a batch.
	MaxBatchSize int `json:"maxBatchSize" description:"The maximum size of a batch. Only used in batch mode."`
	// MaxBatchWait indicates the maximum interval between two batches.
	MaxBatchWait api.Duration `json:"maxBatchWait" description:"The amount of time to wait before force writing the batch that hadn't reached the max size. Only used in batch mode."`

	// ThrottleEnable defines whether throttling will be applied to the batching process.
	ThrottleEnable bool `json:"throttleEnable" description:"Whether batching throttling is enabled. Only used in batch mode."`
	// ThrottleQPS defines the allowed rate of batches per second sent to the delegate backend.
	ThrottleQPS float32 `json:"throttleQPS" description:"Maximum average number of batches per second. "`
	// ThrottleBurst defines the maximum number of requests sent to the delegate backend at the same moment in case
	// the capacity defined by ThrottleQPS was not utilized.
	ThrottleBurst int `json:"throttleBurst" description:"Maximum number of requests sent at the same moment if ThrottleQPS was not utilized before. Only used in batch mode."`

	// Whether the delegate backend should be called asynchronously.
	AsyncDelegate bool `json:"asyncDelegate"`
}

func (c *Config) BatchConfig() *BatchConfig {
	if c == nil {
		return nil
	}
	return &BatchConfig{
		BufferSize:     c.BufferSize,
		MaxBatchSize:   c.MaxBatchSize,
		MaxBatchWait:   c.MaxBatchWait.Duration,
		ThrottleEnable: c.ThrottleEnable,
		ThrottleQPS:    c.ThrottleQPS,
		ThrottleBurst:  c.ThrottleBurst,
		AsyncDelegate:  c.AsyncDelegate,
	}
}
