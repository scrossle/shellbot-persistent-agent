package srv

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"srv.exe.dev/db"
)

type Server struct {
	DB           *sql.DB
	Hostname     string
	TemplatesDir string
	StaticDir    string
	AgentDir     string
}

type pageData struct {
	Hostname    string
	Now         string
	UserEmail   string
	Timers      []timerEntry
	RecentRuns  []runEntry
	DailyLog    string
	LongTerm    string
	Identity    string
	Health      healthInfo
}

type timerEntry struct {
	Name   string
	Next   string
	Last   string
	Passed string
}

type runEntry struct {
	Timestamp string
	Prompt    string
	ConvID    string
}

type healthInfo struct {
	Uptime    string
	Disk      string
	Memory    string
	Load      string
	Shelley   string
	Failed    string
}

func New(dbPath, hostname string) (*Server, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(thisFile)
	home, _ := os.UserHomeDir()
	srv := &Server{
		Hostname:     hostname,
		TemplatesDir: filepath.Join(baseDir, "templates"),
		StaticDir:    filepath.Join(baseDir, "static"),
		AgentDir:     filepath.Join(home, ".agent"),
	}
	if err := srv.setUpDatabase(dbPath); err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *Server) HandleRoot(w http.ResponseWriter, r *http.Request) {
	userEmail := strings.TrimSpace(r.Header.Get("X-ExeDev-Email"))
	now := time.Now()

	data := pageData{
		Hostname:   s.Hostname,
		Now:        now.Format("2006-01-02 15:04 MST"),
		UserEmail:  userEmail,
		Timers:     s.getTimers(),
		RecentRuns: s.getRecentRuns(20),
		DailyLog:   s.readFile(filepath.Join(s.AgentDir, "memory", "daily", now.Format("2006-01-02")+".md")),
		LongTerm:   s.readFile(filepath.Join(s.AgentDir, "memory", "LONGTERM.md")),
		Identity:   s.readFile(filepath.Join(s.AgentDir, "identity.md")),
		Health:     s.getHealth(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderTemplate(w, "dashboard.html", data); err != nil {
		slog.Warn("render template", "url", r.URL.Path, "error", err)
		http.Error(w, "template error", 500)
	}
}

func (s *Server) HandleAPI(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	data := map[string]any{
		"hostname": s.Hostname,
		"time":     now.Format(time.RFC3339),
		"timers":   s.getTimers(),
		"runs":     s.getRecentRuns(20),
		"health":   s.getHealth(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) getTimers() []timerEntry {
	out, err := exec.Command("systemctl", "list-timers", "agent-*", "--no-pager", "--no-legend").Output()
	if err != nil {
		return nil
	}
	var timers []timerEntry
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		// Fields: NEXT(date time tz) LEFT LAST(date time tz or -) PASSED UNIT ACTIVATES
		// Find the ACTIVATES field (last one) and UNIT (second to last)
		unit := fields[len(fields)-2]
		name := strings.TrimSuffix(strings.TrimPrefix(unit, "agent-"), ".timer")

		// Parse NEXT: first 3 fields
		nextStr := strings.Join(fields[0:3], " ")
		if fields[0] == "-" {
			nextStr = "-"
		}

		// LEFT is field[3]
		leftStr := fields[3]
		if len(fields) > 4 && !strings.Contains(fields[3], ":") && fields[3] != "-" {
			// LEFT might be multi-word like "1 day 2h"
			leftStr = fields[3]
		}

		timers = append(timers, timerEntry{
			Name: name,
			Next: nextStr,
			Last: leftStr,
		})
	}
	return timers
}

func (s *Server) getRecentRuns(n int) []runEntry {
	path := filepath.Join(s.AgentDir, "logs", "runs.log")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var runs []runEntry
	// Read last n lines
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}
	for i := len(lines) - 1; i >= start; i-- {
		line := lines[i]
		if line == "" {
			continue
		}
		// Format: TIMESTAMP prompt=NAME conv=ID
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		ts := parts[0]
		var prompt, convID string
		for _, kv := range parts[1:] {
			if strings.HasPrefix(kv, "prompt=") {
				prompt = strings.TrimSuffix(strings.TrimPrefix(kv, "prompt="), ".md")
			} else if strings.HasPrefix(kv, "conv=") {
				convID = strings.TrimPrefix(kv, "conv=")
			}
		}
		runs = append(runs, runEntry{Timestamp: ts, Prompt: prompt, ConvID: convID})
	}
	return runs
}

func (s *Server) getHealth() healthInfo {
	var h healthInfo
	if out, err := exec.Command("uptime", "-p").Output(); err == nil {
		h.Uptime = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("bash", "-c", "df -h / | tail -1 | awk '{print $5 \" used (\" $4 \" free)\"}'").Output(); err == nil {
		h.Disk = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("bash", "-c", "free -h | awk '/^Mem:/ {print $3 \"/\" $2}'").Output(); err == nil {
		h.Memory = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("bash", "-c", "cat /proc/loadavg | awk '{print $1, $2, $3}'").Output(); err == nil {
		h.Load = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("systemctl", "is-active", "shelley.service").Output(); err == nil {
		h.Shelley = strings.TrimSpace(string(out))
	} else {
		h.Shelley = "unknown"
	}
	if out, err := exec.Command("systemctl", "--failed", "--no-pager", "--no-legend").Output(); err == nil {
		s := strings.TrimSpace(string(out))
		if s == "" {
			h.Failed = "none"
		} else {
			h.Failed = s
		}
	}
	return h
}

func (s *Server) readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func (s *Server) renderTemplate(w http.ResponseWriter, name string, data any) error {
	path := filepath.Join(s.TemplatesDir, name)
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		return fmt.Errorf("parse template %q: %w", name, err)
	}
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("execute template %q: %w", name, err)
	}
	return nil
}

func (s *Server) setUpDatabase(dbPath string) error {
	wdb, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}
	s.DB = wdb
	if err := db.RunMigrations(wdb); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func (s *Server) Serve(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.HandleRoot)
	mux.HandleFunc("GET /api/status", s.HandleAPI)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(s.StaticDir))))
	slog.Info("starting server", "addr", addr)
	return http.ListenAndServe(addr, mux)
}
