package filter

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
	Found    int
	NotFound int
}

func FilterPhones(ctx context.Context, client *telegram.Client,
	phoneNumbers []string, outputFile string, maxPerAccount int, taskID string) *Stats {

	api := client.API()
	st := &Stats{}

	for _, phone := range phoneNumbers {
		if maxPerAccount > 0 && (st.Found+st.NotFound) >= maxPerAccount {
			break
		}
		phone = strings.TrimSpace(phone)
		if phone == "" { continue }

		res, err := api.ContactsImportContacts(ctx, []tg.InputPhoneContact{{
			Phone: phone, FirstName: "Check", LastName: "", ClientID: 0,
		}})
		if err != nil { st.NotFound++; continue }

		if len(res.Users) > 0 {
			u, ok := res.Users[0].(*tg.User)
			if !ok { st.NotFound++; continue }
			username := u.Username
			uid := fmt.Sprintf("%d", u.ID)
			line := fmt.Sprintf("%s | @%s | ID:%s\n", phone, username, uid)
			if u.Premium {
				line = fmt.Sprintf("%s | @%s | ID:%s | Premium\n", phone, username, uid)
			}
			f, _ := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			f.WriteString(line)
			f.Close()
			st.Found++
			fmt.Printf("[筛号] √ %s → @%s\n", phone, username)
			api.ContactsDeleteContacts(ctx, []tg.InputUserClass{
				&tg.InputUser{UserID: u.ID, AccessHash: u.AccessHash},
			})
		} else {
			st.NotFound++
			fmt.Printf("[筛号] × %s\n", phone)
		}
		time.Sleep(time.Duration(3+rand.Intn(5)) * time.Second)
	}
	fmt.Printf("[筛号] 完成! 找到:%d 未找到:%d\n", st.Found, st.NotFound)
	return st
}

func GeneratePhoneRange(startPhone string, count int) []string {
	phones := make([]string, 0, count)
	prefix, numStr := "", startPhone
	if idx := strings.IndexAny(startPhone, "0123456789"); idx >= 0 {
		prefix, numStr = startPhone[:idx], startPhone[idx:]
	}
	var base int64
	fmt.Sscanf(numStr, "%d", &base)
	for i := 0; i < count; i++ {
		phones = append(phones, fmt.Sprintf("%s%d", prefix, base+int64(i)))
	}
	return phones
}
