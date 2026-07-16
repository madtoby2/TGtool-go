package clone

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

type Config struct {
	Source      string
	Target      string
	Limit       int
	TextOnly    bool
	MediaOnly   bool
	Forward     bool
	OldestFirst bool
	IntervalMin int
	IntervalMax int
}

type Stats struct {
	Cloned  int
	Skipped int
	Errors  int
}

func CloneChannel(ctx context.Context, client *telegram.Client, cfg Config, taskID string) (*Stats, error) {
	api := client.API()
	st := &Stats{}

	// Resolve source
	srcUsername := strings.TrimPrefix(cfg.Source, "@")
	srcUsername = strings.TrimPrefix(srcUsername, "https://t.me/")
	srcRes, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: srcUsername})
	if err != nil {
		return st, fmt.Errorf("source not found: %w", err)
	}

	var srcPeer tg.InputPeerClass
	for _, chat := range srcRes.Chats {
		if ch, ok := chat.(*tg.Channel); ok {
			srcPeer = &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}
			fmt.Printf("[复刻] 源: %s\n", ch.Title)
		}
	}
	if srcPeer == nil {
		return st, fmt.Errorf("source not a channel")
	}

	// Resolve target
	dstUsername := strings.TrimPrefix(cfg.Target, "@")
	dstUsername = strings.TrimPrefix(dstUsername, "https://t.me/")
	dstRes, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: dstUsername})
	if err != nil {
		return st, fmt.Errorf("target not found: %w", err)
	}

	var dstPeer tg.InputPeerClass
	for _, chat := range dstRes.Chats {
		if ch, ok := chat.(*tg.Channel); ok {
			dstPeer = &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}
			fmt.Printf("[复刻] 目标: %s\n", ch.Title)
		}
	}
	if dstPeer == nil {
		return st, fmt.Errorf("target not a channel")
	}

	// Read messages
	offsetID := 0
	for {
		if cfg.Limit > 0 && st.Cloned >= cfg.Limit {
			break
		}

		msgs, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:      srcPeer,
			Limit:     100,
			OffsetID:  offsetID,
			AddOffset: 0,
		})
		if err != nil {
			break
		}

		var messages []*tg.Message
		switch r := msgs.(type) {
		case *tg.MessagesMessagesSlice:
			for _, m := range r.Messages {
				if msg, ok := m.(*tg.Message); ok {
					messages = append(messages, msg)
				}
			}
		case *tg.MessagesChannelMessages:
			for _, m := range r.Messages {
				if msg, ok := m.(*tg.Message); ok {
					messages = append(messages, msg)
				}
			}
		}

		if len(messages) == 0 {
			break
		}

		for _, msg := range messages {
			if cfg.Limit > 0 && st.Cloned >= cfg.Limit {
				break
			}

			hasMedia := msg.Media != nil

			if cfg.TextOnly && hasMedia {
				st.Skipped++
				continue
			}
			if cfg.MediaOnly && !hasMedia {
				st.Skipped++
				continue
			}

			if cfg.Forward {
				_, err := api.MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
					FromPeer: srcPeer,
					ToPeer:   dstPeer,
					ID:       []int{msg.ID},
				})
				if err == nil {
					st.Cloned++
				} else {
					st.Errors++
				}
			} else {
				_, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
					Peer:    dstPeer,
					Message: msg.Message,
				})
				if err == nil {
					st.Cloned++
				} else {
					st.Errors++
				}
			}

			if st.Cloned%20 == 0 {
				fmt.Printf("[复刻] 进度: %d/%d\n", st.Cloned, cfg.Limit)
			}

			time.Sleep(time.Duration(cfg.IntervalMin+rand.Intn(cfg.IntervalMax-cfg.IntervalMin+1)) * time.Second)
		}

		offsetID = messages[len(messages)-1].ID
		if len(messages) < 100 {
			break
		}
	}

	fmt.Printf("[复刻] 完成! 克隆:%d 跳过:%d 错误:%d\n", st.Cloned, st.Skipped, st.Errors)
	return st, nil
}
