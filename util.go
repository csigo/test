package test

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"text/template"
	"time"
)

const (
	// define min and max port range
	maxPort = 65535
	minPort = 10000
)

var (
	curPort = int32(maxPort + 1)
)

// BookPorts books free portsx
func BookPorts(num int, host ...string) ([]int, error) {
	result := make([]int, 0, num)
	for len(result) < num {
		newPort := atomic.AddInt32(&curPort, -1)
		if newPort < minPort {
			return nil, errors.New("running out of available ports")
		}
		if portAvailable(newPort, host...) {
			result = append(result, int(newPort))
		}
	}
	return result, nil
}

// CheckExecutable checks if given names are executables
func CheckExecutable(names ...string) error {
	for _, name := range names {
		if _, err := exec.LookPath(name); err != nil {
			return err
		}
	}
	return nil
}

// ApplyTemplate applies given variables to the "tplStr" and stores to "filePath"
func ApplyTemplate(filePath string, tplStr string, vars map[string]interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	tpl, err := template.New("tpl").Parse(tplStr)
	if err != nil {
		return err
	}
	return tpl.Execute(file, vars)
}

// CheckListening checks if localhost is listening given ports
func CheckListening(ports ...int) bool {
	for _, p := range ports {
		conn, err := net.DialTimeout("tcp4", fmt.Sprintf("localhost:%d", p), 500*time.Millisecond)
		if err != nil {
			return false
		}
		conn.Close()
	}
	return true
}

// Exec runs "name" and "arg" in directory "workdir" with environements "envs" and wait util
// process finined. It returns error if fail to execute or exit code is not zero.
func Exec(workdir string, envs []string, stdin io.Reader, name string, arg ...interface{}) error {
	argStr := make([]string, 0, len(arg))
	for _, a := range arg {
		argStr = append(argStr, fmt.Sprintf("%v", a))
	}
	cmd := exec.Command(name, argStr...)
	cmd.Env = append(os.Environ(), envs...)
	cmd.Dir = workdir
	if stdin != nil {
		cmd.Stdin = stdin
	}
	bs, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fail to start process, output:%s, err:%v", string(bs), err)
	}
	if !cmd.ProcessState.Success() {
		return fmt.Errorf("fail to start process, output:%s", string(bs))
	}
	return nil
}

// CombineError given errors into one err, note that nil error will be ignored
func CombineError(errs ...error) error {
	result := []string{}
	for _, err := range errs {
		if err != nil {
			result = append(result, err.Error())
		}
	}
	if len(result) != 0 {
		return errors.New(strings.Join(result, ", "))
	}
	return nil
}

func portAvailable(port int32, host ...string) bool {
	// Listen to tcp4 only instead of tcp which may return an ipv6 address port.
	h := ""
	if len(h) != 0 {
		h = host[0]
	}
	l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", h, port))
	if err != nil {
		return false
	}
	l.Close()
	return true
}

// WaitPortAvail waits until the port becomes dialable. It returns error if it's not
// available after timeout.
func WaitPortAvail(port int, timeout time.Duration, host ...string) error {
	h := "localhost"
	if len(host) != 0 {
		h = host[0]
	}
	addr := fmt.Sprintf("%s:%d", h, port)
	timer := time.NewTimer(timeout)
	wait := 1 * time.Second
	for {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			c.Close()
			timer.Stop()
			return nil
		}
		fmt.Printf("Attempt to dial %v failed, err %v\n", addr, err)
		select {
		case <-timer.C:
			return fmt.Errorf("attempt to dial %v timeout after %v", port, timeout)
		case <-time.After(wait):
			fmt.Printf("Do another try after waiting for %v\n", wait)
			wait *= 2
		}
	}
}
