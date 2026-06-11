package platform

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type StreamHandler func(context.Context, redis.XMessage) error

type ConsumeOptions struct {
	Count          int64
	Block          time.Duration
	MinIdle        time.Duration
	ClaimBatchSize int64
}

func ConsumeGroup(ctx context.Context, client *redis.Client, stream, group, consumer string, handler StreamHandler) error {
	return ConsumeGroupWithOptions(ctx, client, stream, group, consumer, handler, ConsumeOptions{
		Count:          50,
		Block:          5 * time.Second,
		MinIdle:        30 * time.Second,
		ClaimBatchSize: 100,
	})
}

func ConsumeGroupWithOptions(ctx context.Context, client *redis.Client, stream, group, consumer string, handler StreamHandler, opts ConsumeOptions) error {
	if opts.Count <= 0 {
		opts.Count = 50
	}
	if opts.Block <= 0 {
		opts.Block = 5 * time.Second
	}
	if opts.MinIdle <= 0 {
		opts.MinIdle = 30 * time.Second
	}
	if opts.ClaimBatchSize <= 0 {
		opts.ClaimBatchSize = 100
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := reclaimPending(ctx, client, stream, group, consumer, handler, opts); err != nil {
			return err
		}

		streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{stream, ">"},
			Count:    opts.Count,
			Block:    opts.Block,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			return fmt.Errorf("xreadgroup %s/%s: %w", stream, group, err)
		}

		for _, strm := range streams {
			if err := handleMessages(ctx, client, stream, group, strm.Messages, handler); err != nil {
				return err
			}
		}
	}
}

func reclaimPending(ctx context.Context, client *redis.Client, stream, group, consumer string, handler StreamHandler, opts ConsumeOptions) error {
	start := "0-0"
	for {
		result, nextStart, err := client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   stream,
			Group:    group,
			Consumer: consumer,
			MinIdle:  opts.MinIdle,
			Start:    start,
			Count:    opts.ClaimBatchSize,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return nil
			}
			return fmt.Errorf("xautoclaim %s/%s: %w", stream, group, err)
		}
		if len(result) == 0 {
			return nil
		}
		if err := handleMessages(ctx, client, stream, group, result, handler); err != nil {
			return err
		}
		if nextStart == "0-0" || nextStart == start {
			return nil
		}
		start = nextStart
	}
}

func handleMessages(ctx context.Context, client *redis.Client, stream, group string, messages []redis.XMessage, handler StreamHandler) error {
	for _, msg := range messages {
		if err := handler(ctx, msg); err != nil {
			continue
		}
		if err := client.XAck(ctx, stream, group, msg.ID).Err(); err != nil {
			return fmt.Errorf("xack %s: %w", stream, err)
		}
	}
	return nil
}
