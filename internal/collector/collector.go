package collector

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

type Stats struct {
	Found   int
	Saved   int
	Skipped int
}

type Collector struct {
	mu     sync.Mutex
	stop   map[string]bool
	stats  map[string]*Stats
}

func New() *Collector {
	return &Collector{
		stop:  make(map[string]bool),
		stats: make(map[string]*Stats),
	}
}

func (c *Collector) Stop(taskID string) {
	c.mu.Lock()
	c.stop[taskID] = true
	c.mu.Unlock()
}

func (c *Collector) shouldStop(taskID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stop[taskID]
}

func (c *Collector) CollectGroupsByKeyword(ctx context.Context, client *telegram.Client,
	keywords []string, outputFile string, minMembers int, saveDetails bool, taskID string) (*Stats, error) {

	c.mu.Lock()
	c.stop[taskID] = false
	c.stats[taskID] = &Stats{}
	c.mu.Unlock()

	api := client.API()
	existing := loadExisting(outputFile)
	saved := 0

	for _, kw := range keywords {
		if c.shouldStop(taskID) {
			break
		}
		fmt.Printf("[采集群组] 关键词: %s\n", kw)

		res, err := api.MessagesSearchGlobal(ctx, &tg.MessagesSearchGlobalRequest{
			Q:      kw,
			Filter: &tg.InputMessagesFilterEmpty{},
			Limit:  50,
		})
		if err != nil {
			continue
		}

		var chats []tg.ChatClass
		switch r := res.(type) {
		case *tg.MessagesMessagesSlice:
			chats = r.Chats
		case *tg.MessagesChannelMessages:
			chats = r.Chats
		}

		for _, chat := range chats {
			if c.shouldStop(taskID) {
				break
			}
			ch, ok := chat.(*tg.Channel)
			if !ok {
				continue
			}
			title := ch.Title
			username := ch.Username
			members := ch.ParticipantsCount

			if ch.Broadcast && !ch.Megagroup {
				c.stats[taskID].Skipped++
				continue
			}
			if minMembers > 0 && members < minMembers {
				c.stats[taskID].Skipped++
				continue
			}

			link := username
			if link == "" {
				link = fmt.Sprintf("https://t.me/c/%d", ch.ID)
			}
			if existing[link] {
				c.stats[taskID].Skipped++
				continue
			}
			existing[link] = true

			f, _ := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if saveDetails {
				fmt.Fprintf(f, "@%s | %s | 人数:%d\n", link, title, members)
			} else {
				fmt.Fprintf(f, "@%s\n", link)
			}
			f.Close()
			c.stats[taskID].Found++
			c.stats[taskID].Saved++
			saved++
			fmt.Printf("[采集群组] +%s (%d人)\n", title, members)
		}
		time.Sleep(time.Duration(3+rand.Intn(5)) * time.Second)
	}
	return c.stats[taskID], nil
}

func (c *Collector) CollectMembers(ctx context.Context, client *telegram.Client,
	groupLink string, outputFile string, maxCount int, saveDetails bool, taskID string) (*Stats, error) {

	c.mu.Lock()
	c.stop[taskID] = false
	c.stats[taskID] = &Stats{}
	c.mu.Unlock()

	api := client.API()
	saved := 0

	username := strings.TrimPrefix(groupLink, "@")
	username = strings.TrimPrefix(username, "https://t.me/")
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
	if err != nil {
		return nil, err
	}

	var channel *tg.InputChannel
	for _, chat := range resolved.Chats {
		if ch, ok := chat.(*tg.Channel); ok {
			channel = &tg.InputChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}
			fmt.Printf("[采集成员] 开始: %s\n", ch.Title)
			break
		}
	}
	if channel == nil {
		return nil, fmt.Errorf("not a group/channel")
	}

	offset := 0
	for {
		if c.shouldStop(taskID) || (maxCount > 0 && saved >= maxCount) {
			break
		}
		res, err := api.ChannelsGetParticipants(ctx, &tg.ChannelsGetParticipantsRequest{
			Channel: channel, Filter: &tg.ChannelParticipantsSearch{Q: ""},
			Offset: offset, Limit: 200,
		})
		if err != nil {
			break
		}

		var users []tg.UserClass
		switch r := res.(type) {
		case *tg.ChannelsChannelParticipants:
			users = r.Users
		}

		for _, u := range users {
			if c.shouldStop(taskID) || (maxCount > 0 && saved >= maxCount) {
				break
			}
			user, ok := u.(*tg.User)
			if !ok || user.Bot {
				continue
			}
			username := user.Username
			uid := fmt.Sprintf("%d", user.ID)
			f, _ := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if saveDetails {
				fmt.Fprintf(f, "%s | %s %s\n", orStr(username, uid), user.FirstName, user.LastName)
			} else {
				fmt.Fprintf(f, "%s\n", orStr(username, uid))
			}
			f.Close()
			saved++
		}
		offset += len(users)
		if len(users) < 200 {
			break
		}
		time.Sleep(time.Duration(1+rand.Intn(3)) * time.Second)
	}

	c.stats[taskID].Saved = saved
	fmt.Printf("[采集成员] 完成! 保存 %d 个成员\n", saved)
	return c.stats[taskID], nil
}

func loadExisting(path string) map[string]bool {
	m := make(map[string]bool)
	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if idx := strings.Index(line, "|"); idx > 0 {
			line = strings.TrimSpace(line[:idx])
		}
		m[strings.TrimPrefix(line, "@")] = true
	}
	return m
}

func orStr(a, b string) string {
	if a != "" { return a }
	return b
}
