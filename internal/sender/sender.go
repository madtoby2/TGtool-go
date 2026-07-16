package sender

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

type Stats struct {
	Sent    int
	Failed  int
	Blocked int
	Already int
	Skipped int
}

type Sender struct {
	mu   sync.Mutex
	stop map[string]bool
}

func New() *Sender {
	return &Sender{stop: make(map[string]bool)}
}

func (s *Sender) Stop(taskID string) {
	s.mu.Lock(); s.stop[taskID] = true; s.mu.Unlock()
}

func (s *Sender) shouldStop(taskID string) bool {
	s.mu.Lock(); defer s.mu.Unlock()
	return s.stop[taskID]
}

func (s *Sender) JoinGroups(ctx context.Context, client *telegram.Client,
	links []string, mode string, intervalMin, intervalMax int, taskID string) *Stats {

	s.mu.Lock(); s.stop[taskID] = false; s.mu.Unlock()
	api := client.API()
	st := &Stats{}

	if mode == "leave" {
		res, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{Limit: 200})
		if err == nil {
			switch r := res.(type) {
			case *tg.MessagesDialogsSlice:
				for _, d := range r.Dialogs {
					if s.shouldStop(taskID) { break }
					dp, ok := d.(*tg.Dialog)
					if !ok { continue }
					if p, ok := dp.Peer.(*tg.PeerChannel); ok {
						api.ChannelsLeaveChannel(ctx, &tg.InputChannel{ChannelID: p.ChannelID})
						st.Sent++
					}
					time.Sleep(time.Duration(intervalMin+rand.Intn(intervalMax-intervalMin+1)) * time.Second)
				}
			}
		}
		return st
	}

	for _, link := range links {
		if s.shouldStop(taskID) { break }
		username := resolveUsername(link)
		if username == "" { st.Failed++; continue }

		// Try invite hash
		if strings.HasPrefix(link, "+") || regexp.MustCompile(`^[a-zA-Z0-9+/=_-]{15,}$`).MatchString(link) {
			hash := strings.TrimPrefix(link, "+")
			if strings.HasPrefix(hash, "https://t.me/+") {
				hash = strings.TrimPrefix(hash, "https://t.me/+")
			}
			_, err := api.MessagesImportChatInvite(ctx, hash)
			if err == nil { st.Sent++; fmt.Printf("[加群] 成功: %s\n", link[:minInt(40, len(link))]) } else { st.Failed++ }
			continue
		}

		res, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil { st.Failed++; continue }

		for _, chat := range res.Chats {
			if ch, ok := chat.(*tg.Channel); ok {
				_, err := api.ChannelsJoinChannel(ctx, &tg.InputChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash})
				if err == nil { st.Sent++; fmt.Printf("[加群] 成功: @%s\n", username) } else { st.Failed++ }
			}
		}
		time.Sleep(time.Duration(intervalMin+rand.Intn(intervalMax-intervalMin+1)) * time.Second)
	}
	return st
}

func (s *Sender) SendToGroups(ctx context.Context, client *telegram.Client,
	messages []string, intervalMin, intervalMax int, autoReply string, taskID string) *Stats {

	s.mu.Lock(); s.stop[taskID] = false; s.mu.Unlock()
	api := client.API()
	st := &Stats{}
	if len(messages) == 0 { return st }

	res, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{Limit: 200})
	if err != nil { return st }

	type groupInfo struct {
		peer tg.InputPeerClass
		name string
	}
	var groups []groupInfo

	switch r := res.(type) {
	case *tg.MessagesDialogsSlice:
		for _, d := range r.Dialogs {
			dp, ok := d.(*tg.Dialog)
			if !ok { continue }
			if p, ok := dp.Peer.(*tg.PeerChannel); ok {
				for _, chat := range r.Chats {
					if ch, ok := chat.(*tg.Channel); ok && ch.ID == p.ChannelID && ch.Megagroup {
						groups = append(groups, groupInfo{
							peer: &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash},
							name: ch.Title,
						})
					}
				}
			}
		}
	}

	for {
		if s.shouldStop(taskID) { break }
		for _, g := range groups {
			if s.shouldStop(taskID) { break }
			msg := messages[rand.Intn(len(messages))]
			_, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{Peer: g.peer, Message: msg})
			if err == nil {
				st.Sent++; fmt.Printf("[群发] → %s\n", msg[:minInt(30, len(msg))])
			} else {
				st.Failed++
			}
			time.Sleep(time.Duration(intervalMin+rand.Intn(intervalMax-intervalMin+1)) * time.Second)
		}
		wait := 300 + rand.Intn(300)
		fmt.Printf("[群发] 一轮完成, %ds后下一轮\n", wait)
		time.Sleep(time.Duration(wait) * time.Second)
	}
	return st
}

func (s *Sender) SendDM(ctx context.Context, client *telegram.Client,
	targets []string, messages []string, maxPerAccount int, intervalMin, intervalMax int, taskID string) *Stats {

	s.mu.Lock(); s.stop[taskID] = false; s.mu.Unlock()
	api := client.API()
	st := &Stats{}

	for _, target := range targets {
		if s.shouldStop(taskID) || (maxPerAccount > 0 && st.Sent >= maxPerAccount) { break }

		username := resolveUsername(target)
		res, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil { st.Failed++; continue }

		var peer tg.InputPeerClass
		for _, u := range res.Users {
			if user, ok := u.(*tg.User); ok {
				peer = &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}
			}
		}
		if peer == nil { st.Failed++; continue }

		msg := messages[rand.Intn(len(messages))]
		_, err = api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{Peer: peer, Message: msg})
		if err == nil { st.Sent++ } else { st.Failed++ }
		time.Sleep(time.Duration(intervalMin+rand.Intn(intervalMax-intervalMin+1)) * time.Second)
	}
	fmt.Printf("[私信] 完成! 成功:%d 失败:%d\n", st.Sent, st.Failed)
	return st
}

func (s *Sender) InviteUsers(ctx context.Context, client *telegram.Client,
	users []string, groupLink string, maxPerAccount int, intervalMin, intervalMax int, taskID string) *Stats {

	s.mu.Lock(); s.stop[taskID] = false; s.mu.Unlock()
	api := client.API()
	st := &Stats{}

	groupUsername := resolveUsername(groupLink)
	gr, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: groupUsername})
	if err != nil { return st }

	var groupPeer *tg.InputPeerChannel
	for _, chat := range gr.Chats {
		if ch, ok := chat.(*tg.Channel); ok {
			groupPeer = &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}
		}
	}
	if groupPeer == nil { return st }

	for _, user := range users {
		if s.shouldStop(taskID) || (maxPerAccount > 0 && st.Sent >= maxPerAccount) { break }

		username := resolveUsername(user)
		ur, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil { st.Failed++; continue }

		var targetUser tg.InputUserClass
		for _, u := range ur.Users {
			if usr, ok := u.(*tg.User); ok {
				targetUser = &tg.InputUser{UserID: usr.ID, AccessHash: usr.AccessHash}
			}
		}
		if targetUser == nil { st.Failed++; continue }

		_, err = api.ChannelsInviteToChannel(ctx, &tg.ChannelsInviteToChannelRequest{
			Channel: &tg.InputChannel{ChannelID: groupPeer.ChannelID, AccessHash: groupPeer.AccessHash},
			Users:   []tg.InputUserClass{targetUser},
		})
		if err == nil { st.Sent++ } else { st.Failed++ }
		time.Sleep(time.Duration(intervalMin+rand.Intn(intervalMax-intervalMin+1)) * time.Second)
	}
	fmt.Printf("[邀请] 完成! 成功:%d 失败:%d\n", st.Sent, st.Failed)
	return st
}

func resolveUsername(link string) string {
	link = strings.TrimSpace(link)
	link = strings.TrimPrefix(link, "@")
	if strings.Contains(link, "t.me/") {
		parts := strings.Split(link, "t.me/")
		link = parts[len(parts)-1]
	}
	if idx := strings.Index(link, "/"); idx > 0 && !strings.HasPrefix(link, "+") {
		link = link[:idx]
	}
	return link
}

func minInt(a, b int) int { if a < b { return a }; return b }
