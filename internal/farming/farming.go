package farming

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

type Stats struct {
	Sent   int
	Failed int
}

// FarmGroups sends scripted messages to groups to simulate activity
func FarmGroups(ctx context.Context, client *telegram.Client,
	groupLinks []string, scripts []string,
	intervalMin, intervalMax int, loopCount int, taskID string) *Stats {

	api := client.API()
	st := &Stats{}

	if len(scripts) == 0 {
		fmt.Println("[炒群] 没有有效话术")
		return st
	}

	// Resolve groups
	var peers []tg.InputPeerClass
	for _, link := range groupLinks {
		username := strings.TrimPrefix(link, "@")
		username = strings.TrimPrefix(username, "https://t.me/")
		res, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil {
			continue
		}
		for _, chat := range res.Chats {
			if ch, ok := chat.(*tg.Channel); ok {
				peers = append(peers, &tg.InputPeerChannel{
					ChannelID:  ch.ID,
					AccessHash: ch.AccessHash,
				})
				// Try to join
				api.ChannelsJoinChannel(ctx, &tg.InputChannel{
					ChannelID:  ch.ID,
					AccessHash: ch.AccessHash,
				})
			}
		}
	}

	if len(peers) == 0 {
		fmt.Println("[炒群] 没有有效目标群组")
		return st
	}

	idx := 0
	loop := 0
	for {
		if loopCount > 0 && loop >= loopCount {
			break
		}
		for _, peer := range peers {
			script := scripts[idx%len(scripts)]
			idx++

			_, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
				Peer:    peer,
				Message: script,
			})
			if err == nil {
				st.Sent++
				fmt.Printf("[炒群] → %s\n", script[:min(30, len(script))])
			} else {
				st.Failed++
			}
			time.Sleep(time.Duration(intervalMin+rand.Intn(intervalMax-intervalMin+1)) * time.Second)
		}
		loop++
		wait := 60 + rand.Intn(240)
		fmt.Printf("[炒群] 第%d轮完成, %ds后下一轮\n", loop, wait)
		time.Sleep(time.Duration(wait) * time.Second)
	}

	return st
}

// CollectScripts reads messages from a group to use as scripts
func CollectScripts(ctx context.Context, client *telegram.Client,
	groupLink string, outputFile string, maxCount int) error {

	username := strings.TrimPrefix(groupLink, "@")
	username = strings.TrimPrefix(username, "https://t.me/")
	api := client.API()

	res, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
	if err != nil {
		return err
	}

	var peer tg.InputPeerClass
	for _, chat := range res.Chats {
		if ch, ok := chat.(*tg.Channel); ok {
			peer = &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}
		}
	}
	if peer == nil {
		return fmt.Errorf("not a channel")
	}

	// Read messages
	msgs, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		Limit: maxCount,
	})
	if err != nil {
		return err
	}

	f, _ := os.Create(outputFile)
	defer f.Close()

	switch r := msgs.(type) {
	case *tg.MessagesMessagesSlice:
		for _, m := range r.Messages {
			if msg, ok := m.(*tg.Message); ok && msg.Message != "" {
				f.WriteString(msg.Message + "\n")
			}
		}
	case *tg.MessagesChannelMessages:
		for _, m := range r.Messages {
			if msg, ok := m.(*tg.Message); ok && msg.Message != "" {
				f.WriteString(msg.Message + "\n")
			}
		}
	}

	fmt.Printf("[采集话术] 完成!\n")
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
