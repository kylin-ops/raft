package health

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/kylin-ops/raft/http/httpclient/grequest"
)

type Checker interface{
	Do()error
}

type Default struct {}

func(*Default)Do()error{
	return nil
}

type Http struct {
	Addr   string
	Method string
	Options *grequest.RequestOptions
}

func (h *Http)Do()error{
	if h.Options.Timeout == 0 {
		h.Options.Timeout = time.Second
	}
	if h.Method == "" {
		h.Method = "GET"
	}
	resp, err := grequest.Request(h.Addr, strings.ToUpper(h.Method), h.Options)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		msg, _ := resp.Text()
		return errors.New(msg)
	}
	return nil
}

type Command struct {
	Command string
	Params  []string
	Timeout time.Duration
}

func (c *Command)Do()error{
	cmd := exec.Command(c.Command, c.Params...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(c.Timeout, func ()  {
		select{
		case <- ctx.Done():
			return
		default:
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	})
	err := cmd.Wait()
	cancel()
	
	return err
}