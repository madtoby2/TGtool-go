package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gotd/td/telegram"

	"github.com/madtoby2/tgtool/internal/clone"
	"github.com/madtoby2/tgtool/internal/collector"
	"github.com/madtoby2/tgtool/internal/config"
	"github.com/madtoby2/tgtool/internal/farming"
	"github.com/madtoby2/tgtool/internal/filter"
	"github.com/madtoby2/tgtool/internal/sender"
	"github.com/madtoby2/tgtool/internal/session"
)

//go:embed web
var webFS embed.FS

var (
	ctx    = context.Background()
	svc    = session.NewManager()
	col    = collector.New()
	snd    = sender.New()

	// task tracking
	taskMu  sync.RWMutex
	tasks   = map[string]*Task{}
	logs    = map[string][]string{}
	sseClients = map[chan string]bool{}
	sseMu     sync.Mutex
)

type Task struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Started string `json:"started"`
	Result  string `json:"result,omitempty"`
}

func main() {
	mux := http.NewServeMux()

	// web dashboard
	webDir, _ := fs.Sub(webFS, "web")
	mux.Handle("/", http.FileServer(http.FS(webDir)))

	// api
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/accounts", handleAccounts)
	mux.HandleFunc("/api/accounts/login", handleLogin)
	mux.HandleFunc("/api/accounts/check", handleCheck)
	mux.HandleFunc("/api/accounts/remove", handleRemove)
	mux.HandleFunc("/api/tasks", handleTasks)
	mux.HandleFunc("/api/tasks/stop", handleStopTask)
	mux.HandleFunc("/api/collect-query", handleCollectQuery)
	mux.HandleFunc("/api/collect-members", handleCollectMembers)
	mux.HandleFunc("/api/join", handleJoin)
	mux.HandleFunc("/api/send", handleSend)
	mux.HandleFunc("/api/dm", handleDM)
	mux.HandleFunc("/api/invite", handleInvite)
	mux.HandleFunc("/api/farm", handleFarm)
	mux.HandleFunc("/api/filter", handleFilter)
	mux.HandleFunc("/api/clone", handleClone)
	mux.HandleFunc("/api/sse", handleSSE)

	port := "9888"
	if p := os.Getenv("PORT"); p != "" { port = p }
	fmt.Printf("╔══════════════════════════════════════╗\n")
	fmt.Printf("║   MADTOBY Cloud Panel  v2.0         ║\n")
	fmt.Printf("║   http://0.0.0.0:%s               ║\n", port)
	fmt.Printf("╚══════════════════════════════════════╝\n")
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// ─── helpers ────────────────────────────────────────────────────────
func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func broadcast(msg string) {
	sseMu.Lock()
	defer sseMu.Unlock()
	for ch := range sseClients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func newTask(name string) string {
	id := fmt.Sprintf("t%d", time.Now().UnixNano()%100000)
	taskMu.Lock()
	tasks[id] = &Task{ID: id, Name: name, Status: "running", Started: time.Now().Format("15:04:05")}
	logs[id] = []string{}
	taskMu.Unlock()
	broadcast(fmt.Sprintf(`{"event":"task","data":{"id":"%s","name":"%s","status":"running"}}`, id, name))
	return id
}

func finishTask(id, status, result string) {
	taskMu.Lock()
	if t, ok := tasks[id]; ok { t.Status = status; t.Result = result }
	taskMu.Unlock()
	broadcast(fmt.Sprintf(`{"event":"task","data":{"id":"%s","status":"%s","result":"%s"}}`, id, status, result))
}

func addLog(id, msg string) {
	taskMu.Lock()
	logs[id] = append(logs[id], fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	if len(logs[id]) > 200 { logs[id] = logs[id][len(logs[id])-200:] }
	taskMu.Unlock()
	broadcast(fmt.Sprintf(`{"event":"log","data":{"task":"%s","msg":"%s"}}`, id, msg))
}

func getClient() *telegram.Client {
	accs := svc.List()
	if len(accs) == 0 { return nil }
	c, err := session.Login(ctx, accs[0].Phone, nil, nil)
	if err != nil { return nil }
	return c
}

// ─── API handlers ───────────────────────────────────────────────────
func handleHealth(w http.ResponseWriter, r *http.Request) {
	accs := svc.List()
	taskMu.RLock(); tcount := len(tasks); taskMu.RUnlock()
	jsonOK(w, map[string]interface{}{
		"status": "ok", "accounts": len(accs), "tasks": tcount,
	})
}

func handleAccounts(w http.ResponseWriter, r *http.Request) {
	accs := svc.List()
	out := make([]map[string]interface{}, 0, len(accs))
	for _, a := range accs {
		out = append(out, map[string]interface{}{
			"phone": a.Phone, "username": a.Username,
			"first_name": a.FirstName, "last_name": a.LastName,
			"premium": a.IsPremium, "active": a.IsActive,
		})
	}
	jsonOK(w, out)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct{ Phone string }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Phone == "" { jsonErr(w, 400, "phone required"); return }

	id := newTask("login: " + req.Phone)
	go func() {
		addLog(id, "sending code to "+req.Phone)
		// In cloud mode, we need the user to input code via a callback
		// For now just connect
		c, err := session.Login(ctx, req.Phone,
			func() string { return "" }, // placeholder — needs interactive
			func() string { return "" },
		)
		if err != nil { finishTask(id, "failed", err.Error()); return }
		if c != nil { finishTask(id, "done", req.Phone); addLog(id, "logged in: "+req.Phone) }
	}()
	jsonOK(w, map[string]string{"task_id": id, "status": "running"})
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
	accs := svc.List()
	type result struct {
		Phone  string `json:"phone"`
		Active bool   `json:"active"`
		User   string `json:"user"`
	}
	var results []result
	for _, a := range accs {
		u := a.Username; if u == "" { u = a.FirstName }
		results = append(results, result{a.Phone, a.IsActive, u})
	}
	jsonOK(w, results)
}

func handleRemove(w http.ResponseWriter, r *http.Request) {
	var req struct{ Phone string }
	json.NewDecoder(r.Body).Decode(&req)
	if svc.Remove(req.Phone) { jsonOK(w, map[string]string{"status": "ok"}) } else { jsonErr(w, 404, "not found") }
}

func handleTasks(w http.ResponseWriter, r *http.Request) {
	taskMu.RLock(); defer taskMu.RUnlock()
	out := make([]*Task, 0, len(tasks))
	for _, t := range tasks { out = append(out, t) }
	jsonOK(w, out)
}

func handleStopTask(w http.ResponseWriter, r *http.Request) {
	var req struct{ ID string }
	json.NewDecoder(r.Body).Decode(&req)
	taskMu.Lock()
	if t, ok := tasks[req.ID]; ok { t.Status = "stopped" }
	taskMu.Unlock()
	col.Stop(req.ID); snd.Stop(req.ID)
	jsonOK(w, map[string]string{"status": "stopped"})
}

func handleCollectQuery(w http.ResponseWriter, r *http.Request) {
	var req struct{ Keyword string }
	json.NewDecoder(r.Body).Decode(&req)
	c := getClient(); if c == nil { jsonErr(w, 400, "no accounts"); return }
	id := newTask("collect: " + req.Keyword)
	out := config.DataDir + "/groups_" + sanitize(req.Keyword) + ".txt"
	go func() {
		addLog(id, "searching: "+req.Keyword)
		st, _ := col.CollectGroupsByKeyword(ctx, c, []string{req.Keyword}, out, 0, true, id)
		result := fmt.Sprintf(`{"saved":%d,"output":"%s"}`, st.Saved, out)
		finishTask(id, "done", result)
	}()
	jsonOK(w, map[string]string{"task_id": id})
}

func handleCollectMembers(w http.ResponseWriter, r *http.Request) {
	var req struct{ Group string }
	json.NewDecoder(r.Body).Decode(&req)
	c := getClient(); if c == nil { jsonErr(w, 400, "no accounts"); return }
	id := newTask("members: " + req.Group)
	out := config.DataDir + "/members_" + sanitize(req.Group) + ".txt"
	go func() {
		addLog(id, "collecting from "+req.Group)
		st, _ := col.CollectMembers(ctx, c, req.Group, out, 0, false, id)
		result := fmt.Sprintf(`{"saved":%d,"output":"%s"}`, st.Saved, out)
		finishTask(id, "done", result)
	}()
	jsonOK(w, map[string]string{"task_id": id})
}

func handleJoin(w http.ResponseWriter, r *http.Request) {
	var req struct{ Links []string; Mode string }
	json.NewDecoder(r.Body).Decode(&req)
	c := getClient(); if c == nil { jsonErr(w, 400, "no accounts"); return }
	if req.Mode == "" { req.Mode = "join" }
	id := newTask("join")
	go func() {
		addLog(id, fmt.Sprintf("%s %d groups", req.Mode, len(req.Links)))
		st := snd.JoinGroups(ctx, c, req.Links, req.Mode, 30, 60, id)
		finishTask(id, "done", fmt.Sprintf("%d done, %d failed", st.Sent, st.Failed))
	}()
	jsonOK(w, map[string]string{"task_id": id})
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	var req struct{ Messages []string }
	json.NewDecoder(r.Body).Decode(&req)
	c := getClient(); if c == nil { jsonErr(w, 400, "no accounts"); return }
	id := newTask("send")
	go func() {
		addLog(id, fmt.Sprintf("sending %d messages", len(req.Messages)))
		st := snd.SendToGroups(ctx, c, req.Messages, 10, 30, "", id)
		finishTask(id, "done", fmt.Sprintf("%d sent", st.Sent))
	}()
	jsonOK(w, map[string]string{"task_id": id})
}

func handleDM(w http.ResponseWriter, r *http.Request) {
	var req struct{ Targets []string; Messages []string; Limit int }
	json.NewDecoder(r.Body).Decode(&req)
	c := getClient(); if c == nil { jsonErr(w, 400, "no accounts"); return }
	if req.Limit == 0 { req.Limit = 30 }
	id := newTask("dm")
	go func() {
		addLog(id, fmt.Sprintf("dm to %d users", len(req.Targets)))
		st := snd.SendDM(ctx, c, req.Targets, req.Messages, req.Limit, 15, 30, id)
		finishTask(id, "done", fmt.Sprintf("%d sent, %d failed", st.Sent, st.Failed))
	}()
	jsonOK(w, map[string]string{"task_id": id})
}

func handleInvite(w http.ResponseWriter, r *http.Request) {
	var req struct{ Users []string; Group string }
	json.NewDecoder(r.Body).Decode(&req)
	c := getClient(); if c == nil { jsonErr(w, 400, "no accounts"); return }
	id := newTask("invite")
	go func() {
		addLog(id, fmt.Sprintf("inviting %d to %s", len(req.Users), req.Group))
		st := snd.InviteUsers(ctx, c, req.Users, req.Group, 50, 30, 60, id)
		finishTask(id, "done", fmt.Sprintf("%d invited", st.Sent))
	}()
	jsonOK(w, map[string]string{"task_id": id})
}

func handleFarm(w http.ResponseWriter, r *http.Request) {
	var req struct{ Scripts []string; Groups []string }
	json.NewDecoder(r.Body).Decode(&req)
	c := getClient(); if c == nil { jsonErr(w, 400, "no accounts"); return }
	id := newTask("farm")
	go func() {
		addLog(id, fmt.Sprintf("farming %d groups", len(req.Groups)))
		st := farming.FarmGroups(ctx, c, req.Groups, req.Scripts, 5, 20, 0, id)
		finishTask(id, "done", fmt.Sprintf("%d messages", st.Sent))
	}()
	jsonOK(w, map[string]string{"task_id": id})
}

func handleFilter(w http.ResponseWriter, r *http.Request) {
	var req struct{ Phones []string; Limit int }
	json.NewDecoder(r.Body).Decode(&req)
	c := getClient(); if c == nil { jsonErr(w, 400, "no accounts"); return }
	if req.Limit == 0 { req.Limit = 30 }
	id := newTask("filter")
	out := fmt.Sprintf("%s/filtered_%d.txt", config.DataDir, len(req.Phones))
	go func() {
		addLog(id, fmt.Sprintf("filtering %d phones", len(req.Phones)))
		st := filter.FilterPhones(ctx, c, req.Phones, out, req.Limit, id)
		finishTask(id, "done", fmt.Sprintf(`{"found":%d,"not_found":%d,"output":"%s"}`, st.Found, st.NotFound, out))
	}()
	jsonOK(w, map[string]string{"task_id": id})
}

func handleClone(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source, Target string; Limit int; Mode string; Forward, Oldest bool
	}
	json.NewDecoder(r.Body).Decode(&req)
	c := getClient(); if c == nil { jsonErr(w, 400, "no accounts"); return }
	id := newTask("clone: " + req.Source + "→" + req.Target)
	go func() {
		addLog(id, "cloning "+req.Source+" → "+req.Target)
		cfg := clone.Config{
			Source: req.Source, Target: req.Target, Limit: req.Limit,
			TextOnly: req.Mode == "text", MediaOnly: req.Mode == "media",
			Forward: !req.Forward || true, IntervalMin: 2, IntervalMax: 8,
		}
		st, _ := clone.CloneChannel(ctx, c, cfg, id)
		finishTask(id, "done", fmt.Sprintf(`{"cloned":%d,"errors":%d}`, st.Cloned, st.Errors))
	}()
	jsonOK(w, map[string]string{"task_id": id})
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok { jsonErr(w, 500, "streaming not supported"); return }

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string, 64)
	sseMu.Lock()
	sseClients[ch] = true
	sseMu.Unlock()

	defer func() {
		sseMu.Lock(); delete(sseClients, ch); sseMu.Unlock()
	}()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func sanitize(s string) string {
	s = strings.Map(func(r rune) rune {
		if utf8.ValidString(string(r)) && r != '/' && r != '\\' && r != ':' && r != '@' { return r }
		return '_'
	}, s)
	if len(s) > 30 { s = s[:30] }
	return s
}

// silence unused imports
var _ = strconv.Itoa
