package main

import (
	"fmt"
	"github.com/shirou/gopsutil/process"
	"golang.org/x/sys/unix"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

type jvmProcess struct {
	pid         int32
	uid         int32
	guid        int32
	currentUid  int32
	currentGuid int32
}

func executeCommand(pid int, command string, arg string) ([]byte, error) {
	jvmProcess, err := checkPermissions(int32(pid))
	if err != nil {
		return nil, err
	}

	err = jvmProcess.checkPermissions()
	if err != nil {
		return nil, err
	}

	if !jvmProcess.checkSocket() {
		err = jvmProcess.createAttachFile()
		if err != nil {
			return nil, err
		}
	}

	err = jvmProcess.sendSIGQUIT()
	if err != nil {
		return nil, err
	}

	fd, err := jvmProcess.connectSocket()
	if err != nil {
		return nil, err
	}

	response, e := writeRequest(command, arg, fd, err)

	return response, e
}

func writeRequest(command string, arg string, fd int, e error) ([]byte, error) {
	//<ver>0<cmd>0<arg>0<arg>0<arg>0
	//see corretto-8/src/hotspot/src/os/linux/vm/attachListener_linux.cpp:231
	//LinuxAttachListener::read_request
	request := make([]byte, 0)
	request = append(request, byte('1'))
	request = append(request, byte(0))

	request = append(request, []byte(command)...)
	request = append(request, byte(0))

	request = append(request, []byte(arg)...)
	request = append(request, byte(0))

	request = append(request, []byte("")...)
	request = append(request, byte(0))

	request = append(request, []byte("")...)
	request = append(request, byte(0))

	unix.Write(fd, request)

	response := make([]byte, 0)

	buf := make([]byte, 8192)
	n, _ := unix.Read(fd, buf)
	defer unix.Close(fd)

	for n != 0 {
		response = append(response, buf...)
		n, e = unix.Read(fd, buf)
	}

	return response, e
}

func (p jvmProcess) checkSocket() bool {
	socketPath := fmt.Sprintf("%s/.java_pid%d", p.getTempPath(), p.pid)

	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return false
	}

	return true
}

func (p jvmProcess) getTempPath() string {
	return fmt.Sprintf("/proc/%v/root/tmp", p.pid)
}

//TODO
// Dynamic attach is allowed only for the clients with the same euid/egid.
// If we are running under root, switch to the required euid/egid automatically.
func (p jvmProcess) checkPermissions() error {
	if p.currentUid != p.uid {
		return fmt.Errorf("uids aren't equal, %v %v ", p.currentGuid, p.guid)
	}

	if p.currentGuid != p.guid {
		return fmt.Errorf("guids aren't equal, %v %v ", p.currentGuid, p.guid)
	}

	return nil
}

func (p jvmProcess) createAttachFile() error {
	attachFile := fmt.Sprintf("%s/.attach_pid%d", p.getTempPath(), p.pid)
	_, e := os.Create(attachFile)
	if e != nil {
		return e
	}
	return nil
}

func (p jvmProcess) sendSIGQUIT() error {
	findProcess, e := os.FindProcess(int(p.pid))
	if e != nil {
		return fmt.Errorf("cannot find process by pid %v %v", p.pid, e)
	}

	e = findProcess.Signal(syscall.SIGQUIT)
	if e != nil {
		return fmt.Errorf("cannot send SIGQUIT to %v %v", p.pid, e)
	}

	sleep := 20 * time.Millisecond
	for {
		time.Sleep(sleep)
		if !p.checkSocket() {
			sleep = 2 * sleep
		} else {
			break
		}
	}

	return nil
}

func (p jvmProcess) connectSocket() (int, error) {
	socketPath := fmt.Sprintf("%s/.java_pid%d", p.getTempPath(), p.pid)

	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return -1, err
	}

	addr := unix.SockaddrUnix{
		Name: socketPath,
	}

	err = unix.Connect(fd, &addr)
	if err != nil {
		return -1, err
	}

	return fd, nil
}

func checkPermissions(pid int32) (jvmProcess, error) {
	processInfo := jvmProcess{}

	processes, err := process.Processes()

	if err != nil {
		return processInfo, err
	}

	current, err := user.Current()

	if err != nil {
		return processInfo, err
	}

	for _, p := range processes {
		if p.Pid == pid {
			processInfo.pid = pid

			//set GID
			gids, err := p.Gids()
			if err != nil {
				return processInfo, err
			}
			processInfo.guid = gids[0]

			//set UID
			uids, err := p.Uids()
			if err != nil {
				return processInfo, err
			}
			processInfo.uid = uids[1]

			//set current GID
			gid, _ := strconv.Atoi(current.Gid)
			processInfo.currentGuid = int32(gid)

			//set current UID
			uid, _ := strconv.Atoi(current.Uid)
			processInfo.currentUid = int32(uid)

			return processInfo, nil
		}
	}

	return processInfo, fmt.Errorf("cannot find process with given id")
}

