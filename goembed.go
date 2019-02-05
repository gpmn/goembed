package goembed

import (
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"qlang.io/cl/qlang"
	"qlang.io/lib/terminal"

	"github.com/kr/pty"
	qipt "qlang.io/cl/interpreter"
	qall "qlang.io/lib/qlang.all"

	sshterm "golang.org/x/crypto/ssh/terminal"
)

var (
	historyFile = os.Getenv("HOME") + "/.qlang.history"
)

// GoEmbed :
type GoEmbed struct {
	lastConn net.Conn
	lock     sync.Mutex
	Pty      *os.File
	Tty      *os.File
	logFile  *os.File
}

func (ge *GoEmbed) startTransfer() {
	go func() {
		buf := make([]byte, 1024)
		for {
			ge.lock.Lock()
			if ge.lastConn == nil {
				ge.lock.Unlock()
				time.Sleep(1 * time.Second)
				continue
			}
			conn := ge.lastConn
			ge.lock.Unlock()

			cnt, err := conn.Read(buf)
			if err != nil {
				log.Printf("GoEmbed.handleConnection - conn.Read %d byte, error %v", cnt, err)
				conn.Close()
				ge.lock.Lock()
				if ge.lastConn == conn {
					ge.lastConn = nil
				}
				ge.lock.Unlock()
				continue
			}
			if ge.logFile != nil {
				ge.logFile.Write(buf[:cnt])
			}

			cw, err := ge.Pty.Write(buf[:cnt])
			if err != nil {
				log.Printf("GoEmbed.handleConnection - ge.Pty.Write(buf[:%d]) write %d bytes, error : %v", cnt, cw, err)
				continue
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			cnt, err := ge.Pty.Read(buf)
			if err != nil {
				log.Println("GoEmbed.handleConnection - ge.Pty.Read(buf) failed : %v", err)
				continue
			}

			if ge.logFile != nil {
				ge.logFile.Write(buf[:cnt])
			}

			ge.lock.Lock()
			if ge.lastConn == nil {
				ge.lock.Unlock()
				time.Sleep(1 * time.Second)
				continue
			}
			conn := ge.lastConn
			ge.lock.Unlock()

			cw, err := conn.Write(buf[:cnt])
			if err != nil {
				log.Printf("GoEmbed.handleConnection - conn.Write(buf[:%d]) write %d bytes, error : %v",
					cnt, cw, err)
				conn.Close()
				ge.lock.Lock()
				if ge.lastConn == conn {
					ge.lastConn = nil
				}
				ge.lock.Unlock()
				continue
			}
		}
	}()
}

// Serve :
func (ge *GoEmbed) Serve(servAddr, logPath string) (err error) {
	if logPath != "" {
		logF, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if nil != err {
			log.Printf("GoEmbed.Serve - os.OpenFile(%s) as log file failed : %v", logPath, err)
			return err
		}
		ge.logFile = logF
	}

	// 准备pty设备
	ge.Pty, ge.Tty, err = pty.Open()
	if nil != err {
		log.Printf("GoEmbed.Serve - failed : %v", err)
		return err
	}

	if _, err = sshterm.MakeRaw(int(ge.Tty.Fd())); nil != err {
		log.Printf("GoEmbed.Serve - sshterm.MakeRaw for ge.Tty failed : %v", err)
	}
	if _, err = sshterm.MakeRaw(int(ge.Pty.Fd())); nil != err {
		log.Printf("GoEmbed.Serve - sshterm.MakeRaw for ge.Tty failed : %v", err)
	}
	// 重定向本进程的输入输出
	syscall.Dup2(int(ge.Tty.Fd()), 0)
	syscall.Dup2(int(ge.Tty.Fd()), 1)
	syscall.Dup2(int(ge.Tty.Fd()), 2)

	ge.startTransfer()

	// 监听
	log.Printf("will serve at address %s", servAddr)
	ln, err := net.Listen("tcp", servAddr)
	if err != nil {
		log.Println("GoEmbed.Serve - tls.Listen @%s failed : %v", servAddr, err)
		return err
	}
	go func(ln net.Listener) {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("GoEmbed.Serve - ln.Accept failed : %v", err)
				continue
			}
			ge.lock.Lock()
			if ge.lastConn != nil {
				ge.lastConn.Close()
			}
			ge.lastConn = conn
			ge.lock.Unlock()
		}
	}(ln)
	go qShell()
	return nil
}

// QShell :
func qShell() {
	qall.InitSafe(false)
	qlang.Import("", qipt.Exports)
	qlang.Import("qlang", qlang.Exports)
	qlang.SetDumpCode("")

	lang := qlang.New()

	// interpreter
	log.Println("qshell in serving now")

	var ret interface{}
	qlang.SetOnPop(func(v interface{}) {
		ret = v
	})

	var tokener tokener
	term := terminal.New(">>> ", "... ", tokener.ReadMore)
	// term.SetWordCompleter(func(line string, pos int) (head string, completions []string, tail string) {
	// 	return line[:pos], []string{"  "}, line[pos:]
	// })

	term.LoadHistroy(historyFile) // load/save histroy
	defer term.SaveHistroy(historyFile)

	for {
		expr, err := term.Scan()
		if err != nil {
			if err == terminal.ErrPromptAborted {
				continue
			}
			log.Printf("GoEmbed.Serve - term.Scan faield %v", err)
			continue
		}
		expr = strings.TrimSpace(expr)
		if expr == "" {
			continue
		}
		err = lang.SafeEval(expr)
		if err != nil {
			log.Printf("GoEmbed.Serve - lang.SafeEval failed, expr %s, err : %v ", expr, err)
			continue
		}
	}
}

// -----------------------------------------------------------------------------

type tokener struct {
	level int
	instr bool
}

var dontReadMoreChars = "+-})];"
var puncts = "([=,*/%|&<>^.:"

func readMore(line string) bool {

	n := len(line)
	if n == 0 {
		return false
	}

	pos := strings.IndexByte(dontReadMoreChars, line[n-1])
	if pos == 0 || pos == 1 {
		return n >= 2 && line[n-2] != dontReadMoreChars[pos]
	}
	return pos < 0 && strings.IndexByte(puncts, line[n-1]) >= 0
}

func findEnd(line string, c byte) int {

	for i := 0; i < len(line); i++ {
		switch line[i] {
		case c:
			return i
		case '\\':
			i++
		}
	}
	return -1
}

func (p *tokener) ReadMore(expr string, line string) (string, bool) { // read more line check

	ret := expr + line + "\n"
	for {
		if p.instr {
			pos := strings.IndexByte(line, '`')
			if pos < 0 {
				return ret, true
			}
			line = line[pos+1:]
			p.instr = false
		}

		pos := strings.IndexAny(line, "{}`'\"")
		if pos < 0 {
			if p.level != 0 {
				return ret, true
			}
			line = strings.TrimRight(line, " \t")
			return ret, readMore(line)
		}
		switch c := line[pos]; c {
		case '{':
			p.level++
		case '}':
			p.level--
		case '`':
			p.instr = true
		default:
			line = line[pos+1:]
			pos = findEnd(line, c)
			if pos < 0 {
				return ret, p.level != 0
			}
		}
		line = line[pos+1:]
	}
}