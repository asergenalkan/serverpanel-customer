package api

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/asergenalkan/serverpanel/internal/config"
	"github.com/asergenalkan/serverpanel/internal/database"
	"github.com/creack/pty"
	"github.com/gofiber/contrib/websocket"
	"github.com/golang-jwt/jwt/v5"
)

// TerminalSession represents an active terminal session
type TerminalSession struct {
	ID       string
	UserID   int64
	Username string
	Role     string
	PTY      *os.File
	Cmd      *exec.Cmd
	Done     chan struct{}
}

// TerminalManager manages terminal sessions
type TerminalManager struct {
	sessions map[string]*TerminalSession
	mu       sync.RWMutex
}

var terminalManager = &TerminalManager{
	sessions: make(map[string]*TerminalSession),
}

// HandleTerminalWebSocket handles terminal WebSocket connections
func (h *Handler) HandleTerminalWebSocket(c *websocket.Conn) {
	defer c.Close()

	// Get token from query
	token := c.Query("token")
	if token == "" {
		c.WriteMessage(websocket.TextMessage, []byte("\r\n\033[31mYetkilendirme hatası: Token gerekli\033[0m\r\n"))
		return
	}

	// Validate token
	claims, err := h.validateTerminalToken(token)
	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte("\r\n\033[31mYetkilendirme hatası: Geçersiz token\033[0m\r\n"))
		return
	}

	userID := int64(claims["user_id"].(float64))
	role := claims["role"].(string)

	// Get username from database
	var username string
	err = h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte("\r\n\033[31mKullanıcı bulunamadı\033[0m\r\n"))
		return
	}

	// Determine shell and working directory based on role
	var shell string
	var homeDir string
	var shellArgs []string

	if role == "admin" {
		// Admin gets root shell
		shell = "/bin/bash"
		homeDir = "/root"
		shellArgs = []string{"-l"}
	} else {
		// Regular users get their own shell in their home directory
		shell = "/bin/bash"
		homeDir = "/home/" + username
		shellArgs = []string{"-l"}

		// Check if home directory exists
		if _, err := os.Stat(homeDir); os.IsNotExist(err) {
			c.WriteMessage(websocket.TextMessage, []byte("\r\n\033[31mKullanıcı dizini bulunamadı\033[0m\r\n"))
			return
		}
	}

	// Create command
	cmd := exec.Command(shell, shellArgs...)
	cmd.Dir = homeDir

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"HOME="+homeDir,
		"USER="+username,
		"SHELL="+shell,
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
	)

	// If not admin, run as the user
	if role != "admin" {
		// Get user UID and GID
		userInfo, err := exec.Command("id", "-u", username).Output()
		if err == nil {
			var uid, gid uint32
			// Parse UID
			for _, b := range userInfo {
				if b >= '0' && b <= '9' {
					uid = uid*10 + uint32(b-'0')
				} else {
					break
				}
			}

			gidInfo, err := exec.Command("id", "-g", username).Output()
			if err == nil {
				for _, b := range gidInfo {
					if b >= '0' && b <= '9' {
						gid = gid*10 + uint32(b-'0')
					} else {
						break
					}
				}

				cmd.SysProcAttr = &syscall.SysProcAttr{
					Credential: &syscall.Credential{
						Uid: uid,
						Gid: gid,
					},
				}
			}
		}
	}

	// Start PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte("\r\n\033[31mTerminal başlatılamadı: "+err.Error()+"\033[0m\r\n"))
		return
	}
	defer ptmx.Close()

	// Set initial size
	pty.Setsize(ptmx, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})

	// Create session
	session := &TerminalSession{
		ID:       fmt.Sprintf("term-%d", time.Now().UnixNano()),
		UserID:   userID,
		Username: username,
		Role:     role,
		PTY:      ptmx,
		Cmd:      cmd,
		Done:     make(chan struct{}),
	}

	terminalManager.mu.Lock()
	terminalManager.sessions[session.ID] = session
	terminalManager.mu.Unlock()

	defer func() {
		terminalManager.mu.Lock()
		delete(terminalManager.sessions, session.ID)
		terminalManager.mu.Unlock()
		close(session.Done)
		cmd.Process.Kill()
	}()

	// Welcome message
	welcomeMsg := "\033[32m"
	if role == "admin" {
		welcomeMsg += "╔════════════════════════════════════════════╗\r\n"
		welcomeMsg += "║     ServerPanel Terminal (Admin)           ║\r\n"
		welcomeMsg += "╚════════════════════════════════════════════╝\r\n"
	} else {
		welcomeMsg += "╔════════════════════════════════════════════╗\r\n"
		welcomeMsg += "║     ServerPanel Terminal (" + username + ")\r\n"
		welcomeMsg += "╚════════════════════════════════════════════╝\r\n"
	}
	welcomeMsg += "\033[0m"
	c.WriteMessage(websocket.BinaryMessage, []byte(welcomeMsg))

	// Channel for PTY output
	outputDone := make(chan struct{})

	// Read from PTY and send to WebSocket
	go func() {
		defer close(outputDone)
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				if err := c.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
		}
	}()

	// Read from WebSocket and send to PTY
	go func() {
		for {
			msgType, msg, err := c.ReadMessage()
			if err != nil {
				cmd.Process.Kill()
				return
			}

			if msgType == websocket.BinaryMessage || msgType == websocket.TextMessage {
				// Check for resize message
				if len(msg) > 0 && msg[0] == '\x01' {
					// Resize message format: \x01<rows>,<cols>
					var rows, cols uint16
					if _, err := parseSize(msg[1:], &rows, &cols); err == nil {
						pty.Setsize(ptmx, &pty.Winsize{
							Rows: rows,
							Cols: cols,
						})
					}
					continue
				}

				// Write to PTY
				ptmx.Write(msg)
			}
		}
	}()

	// Wait for PTY to close or WebSocket to disconnect
	select {
	case <-outputDone:
	case <-session.Done:
	}
}

// validateTerminalToken validates JWT token for terminal
func (h *Handler) validateTerminalToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Check expiration
		if exp, ok := claims["exp"].(float64); ok {
			if time.Unix(int64(exp), 0).Before(time.Now()) {
				return nil, jwt.ErrTokenExpired
			}
		}
		return claims, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}

// parseSize parses terminal size from message
func parseSize(data []byte, rows, cols *uint16) (int, error) {
	if len(data) < 3 {
		return 0, nil
	}

	// Parse rows,cols format
	var r, c uint16
	i := 0
	for ; i < len(data) && data[i] != ','; i++ {
		if data[i] >= '0' && data[i] <= '9' {
			r = r*10 + uint16(data[i]-'0')
		}
	}
	i++ // skip comma
	for ; i < len(data); i++ {
		if data[i] >= '0' && data[i] <= '9' {
			c = c*10 + uint16(data[i]-'0')
		}
	}

	if r > 0 && c > 0 {
		*rows = r
		*cols = c
	}

	return i, nil
}

// HandleTerminalWebSocketDirect returns a WebSocket handler for terminal
func HandleTerminalWebSocketDirect(db *database.DB, cfg *config.Config) func(*websocket.Conn) {
	h := &Handler{db: db, cfg: cfg}
	return h.HandleTerminalWebSocket
}

// setWinsize sets the terminal window size (for systems without TIOCSWINSZ)
func setWinsize(fd uintptr, w, h uint32) {
	ws := &struct {
		Height uint16
		Width  uint16
		x      uint16
		y      uint16
	}{
		Height: uint16(h),
		Width:  uint16(w),
	}
	syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
}
