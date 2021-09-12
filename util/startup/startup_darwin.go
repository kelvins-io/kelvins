package startup

import (
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"syscall"
)

func execProcessCmd(pid int, upType startUpType) (next bool, err error) {
	switch upType {
	case startUpReStart:
		logging.Infof("process %d restart...\n", pid)
		err = processControl(pid, syscall.SIGUSR1)
		logging.Infof("process %d restart over\n", pid)
	case startUpStop:
		logging.Infof("process %d stop...\n", pid)
		err = processControl(pid, syscall.SIGTERM)
		logging.Infof("process %d stop over\n", pid)
	default:
		next = true
	}
	return
}
