//go:build windows

package sys

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

var (
	taskkillPath     string
	taskkillPathOnce sync.Once
)

const (
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE = 0x2000
	JOB_OBJECT_LIMIT_BREAKAWAY_OK      = 0x800
)

type JOBOBJECT_BASIC_LIMIT_INFORMATION struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type JOBOBJECT_EXTENDED_LIMIT_INFORMATION struct {
	BasicLimitInformation JOBOBJECT_BASIC_LIMIT_INFORMATION
	IoInfo                struct {
		ReadOperationCount  uint64
		WriteOperationCount uint64
		ReadTransferCount   uint64
		WriteTransferCount  uint64
	}
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

var (
	kernel32DLL              *syscall.LazyDLL
	createJobObjectW         *syscall.LazyProc
	setInformationJobObject  *syscall.LazyProc
	assignProcessToJobObject *syscall.LazyProc
	jobObjectInitOnce        sync.Once
	jobObjectInitErr         error
)

func initJobObjectAPI() error {
	jobObjectInitOnce.Do(func() {
		kernel32DLL = syscall.NewLazyDLL("kernel32.dll")

		createJobObjectW = kernel32DLL.NewProc("CreateJobObjectW")
		if createJobObjectW == nil {
			jobObjectInitErr = fmt.Errorf("failed to load CreateJobObjectW")
			return
		}

		setInformationJobObject = kernel32DLL.NewProc("SetInformationJobObject")
		if setInformationJobObject == nil {
			jobObjectInitErr = fmt.Errorf("failed to load SetInformationJobObject")
			return
		}

		assignProcessToJobObject = kernel32DLL.NewProc("AssignProcessToJobObject")
		if assignProcessToJobObject == nil {
			jobObjectInitErr = fmt.Errorf("failed to load AssignProcessToJobObject")
			return
		}
	})
	return jobObjectInitErr
}

// SetupCmdSysProcAttr creates a Job Object and configures the command.
// Returns the job handle for later use in AssignProcessToJob and KillProcessGroup.
func SetupCmdSysProcAttr(cmd *exec.Cmd) (syscall.Handle, error) {
	// Set Windows-specific process attributes
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | syscall.CREATE_BREAKAWAY_FROM_JOB,
	}

	jobHandle, err := CreateJobObject()
	if err != nil {
		return 0, err
	}
	// Job Object assignment will be done after cmd.Start() in pool.go
	return jobHandle, nil
}

func CreateJobObject() (syscall.Handle, error) {
	if err := initJobObjectAPI(); err != nil {
		return 0, fmt.Errorf("job object API not available: %w", err)
	}

	handle, _, err := createJobObjectW.Call(0, 0)
	if handle == 0 {
		return 0, fmt.Errorf("CreateJobObject failed: %w", err)
	}

	info := JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | JOB_OBJECT_LIMIT_BREAKAWAY_OK,
		},
	}

	ret, _, err := setInformationJobObject.Call(
		uintptr(handle),
		uintptr(9),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if ret == 0 {
		_ = syscall.CloseHandle(syscall.Handle(handle))
		return 0, fmt.Errorf("SetInformationJobObject failed: %w", err)
	}

	return syscall.Handle(handle), nil
}

func AssignProcessToJob(jobHandle syscall.Handle, process *os.Process) error {
	if err := initJobObjectAPI(); err != nil {
		return fmt.Errorf("job object API not available: %w", err)
	}

	// Open process handle - AssignProcessToJobObject requires a HANDLE, not a PID
	processHandle, err := syscall.OpenProcess(syscall.PROCESS_SET_QUOTA|syscall.PROCESS_TERMINATE, false, uint32(process.Pid))
	if err != nil {
		return fmt.Errorf("failed to open process handle: %w", err)
	}
	defer syscall.CloseHandle(processHandle) //nolint:errcheck

	ret, _, err := assignProcessToJobObject.Call(
		uintptr(jobHandle),
		uintptr(processHandle),
	)
	if ret == 0 {
		return fmt.Errorf("AssignProcessToJobObject failed: %w", err)
	}
	return nil
}

func KillProcessGroup(cmd *exec.Cmd, jobHandle syscall.Handle) {
	if cmd == nil {
		return
	}

	// First, try to close the Job Object (triggers KILL_ON_JOB_CLOSE)
	if jobHandle != 0 {
		_ = syscall.CloseHandle(jobHandle)
	}

	// If process still exists, use taskkill as fallback
	if cmd.Process != nil {
		taskkillPathOnce.Do(func() {
			var err error
			taskkillPath, err = exec.LookPath("taskkill")
			if err != nil {
				taskkillPath = os.Getenv("SystemRoot") + "\\system32\\taskkill.exe"
			}
		})

		killCmd := exec.Command(taskkillPath, "/F", "/T", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
		_ = killCmd.Run()
		_ = cmd.Process.Kill()
	}
}

// CloseJobHandle safely closes a Windows Job Object handle.
func CloseJobHandle(jobHandle syscall.Handle) {
	if jobHandle != 0 {
		_ = syscall.CloseHandle(jobHandle)
	}
}

func IsProcessAlive(process *os.Process) bool {
	if process == nil {
		return false
	}
	// Try to get process exit code to check if it's still running
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(process.Pid))
	if err != nil {
		return false // Process doesn't exist or can't be accessed
	}
	defer syscall.CloseHandle(handle) //nolint:errcheck

	var exitCode uint32
	if err := syscall.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == syscall.STILL_ACTIVE
}
