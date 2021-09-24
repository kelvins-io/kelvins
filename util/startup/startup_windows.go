package startup

import (
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"runtime"
	"syscall"
)

func execProcessCmd(pid int, upType startUpType) (next bool, err error) {
	switch upType {
	case startUpReStart:
		logging.Infof("process platform(%s) not support restart\n", runtime.GOOS)
	case startUpStop:
		logging.Infof("process %d stop...\n", pid)
		err = processControl(pid, syscall.SIGTERM)
		logging.Infof("process %d stop over\n", pid)
	default:
		next = true
	}
	return
}
