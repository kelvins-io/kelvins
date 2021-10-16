package app

import (
	"bytes"
	"context"
	"fmt"
	"gitee.com/kelvins-io/common/convert"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/common/queue"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/util/kprocess"
	"github.com/RichardKnop/machinery/v1"
	queueLog "github.com/RichardKnop/machinery/v1/log"
	"time"
)

// RunQueueApplication runs queue application.
func RunQueueApplication(application *kelvins.QueueApplication) {
	if application == nil || application.Application == nil {
		panic("queueApplication is nil or application is nil")
	}
	// app instance once validate
	{
		err := appInstanceOnceValidate()
		if err != nil {
			logging.Fatal(err.Error())
		}
	}

	// type instance vars
	application.Type = kelvins.AppTypeQueue
	kelvins.QueueAppInstance = application

	err := runQueue(application)
	if err != nil {
		logging.Infof("queueApp runQueue err: %v\n", err)
	}

	appPrepareForceExit()
	// Wait for connections to drain.
	err = appShutdown(application.Application, 0)
	if err != nil {
		logging.Infof("queueApp appShutdown err: %v\n", err)
	}
}

var queueToWorker = map[*queue.MachineryQueue][]*machinery.Worker{}

// runQueue runs queue application.
func runQueue(queueApp *kelvins.QueueApplication) error {
	// 1. init application
	var err error
	err = initApplication(queueApp.Application)
	if err != nil {
		return err
	}
	if !appProcessNext {
		return err
	}

	// 2 init queue vars
	err = setupQueueVars(queueApp)
	if err != nil {
		return err
	}

	// 3. event server
	if kelvins.EventServerAliRocketMQ != nil {
		logging.Info("queueApp start event server ")
		if queueApp.RegisterEventProducer != nil {
			appRegisterEventProducer(queueApp.RegisterEventProducer, queueApp.Type)
		}
		if queueApp.RegisterEventHandler != nil {
			appRegisterEventHandler(queueApp.RegisterEventHandler, queueApp.Type)
		}
	}

	// 4. queue server
	logging.Info("queueApp start queue server consume")
	concurrency := len(queueApp.GetNamedTaskFuncs())
	if kelvins.QueueServerSetting != nil {
		concurrency = kelvins.QueueServerSetting.WorkerConcurrency
	}
	logging.Infof("queueApp count of worker goroutine: %d\n", concurrency)
	consumerTag := queueApp.Application.Name + convert.Int64ToStr(time.Now().Local().UnixNano())

	kp := new(kprocess.KProcess)
	_, err = kp.Listen("", "", kelvins.PIDFile)
	if err != nil {
		return fmt.Errorf("kprocess listen pidFile(%v) err: %v", kelvins.PIDFile, err)
	}
	var queueList []string
	queueList = append(queueList, kelvins.QueueServerSetting.CustomQueueList...)
	errorsChanSize := 0
	if kelvins.QueueRedisSetting != nil && !kelvins.QueueRedisSetting.DisableConsume {
		errorsChanSize += len(queueList)
	}
	if kelvins.QueueAMQPSetting != nil && !kelvins.QueueAMQPSetting.DisableConsume {
		errorsChanSize += len(queueList)
	}
	if kelvins.QueueAliAMQPSetting != nil && !kelvins.QueueAliAMQPSetting.DisableConsume {
		errorsChanSize += len(queueList)
	}
	errorsChan := make(chan error, errorsChanSize)
	for _, customQueue := range queueList {
		cTag := consumerTag
		if len(customQueue) > 0 {
			cTag = customQueue + "-" + consumerTag
		}
		if kelvins.QueueRedisSetting != nil && !kelvins.QueueRedisSetting.DisableConsume && kelvins.QueueServerRedis != nil {
			logging.Infof("queueApp queueServerRedis Consumer Tag: %s\n", cTag)
			worker := kelvins.QueueServerRedis.TaskServer.NewCustomQueueWorker(cTag, concurrency, customQueue)
			worker.LaunchAsync(errorsChan)
			queueToWorker[kelvins.QueueServerRedis] = append(queueToWorker[kelvins.QueueServerRedis], worker)
		}
		if kelvins.QueueAMQPSetting != nil && !kelvins.QueueAMQPSetting.DisableConsume && kelvins.QueueServerAMQP != nil {
			logging.Infof("queueApp queueServerAMQP Consumer Tag: %s\n", cTag)
			worker := kelvins.QueueServerAMQP.TaskServer.NewCustomQueueWorker(cTag, concurrency, customQueue)
			worker.LaunchAsync(errorsChan)
			queueToWorker[kelvins.QueueServerAMQP] = append(queueToWorker[kelvins.QueueServerAMQP], worker)
		}
		if kelvins.QueueAliAMQPSetting != nil && !kelvins.QueueAliAMQPSetting.DisableConsume && kelvins.QueueServerAliAMQP != nil {
			logging.Infof("queueApp queueServerAliAMQP Consumer Tag: %s\n", cTag)
			worker := kelvins.QueueServerAliAMQP.TaskServer.NewCustomQueueWorker(cTag, concurrency, customQueue)
			worker.LaunchAsync(errorsChan)
			queueToWorker[kelvins.QueueServerAliAMQP] = append(queueToWorker[kelvins.QueueServerAliAMQP], worker)
		}
	}
	queueApp.QueueServerToWorker = queueToWorker
	<-kp.Exit() // worker not listen Interrupt,SIGTERM signal stop

	// 5 close
	queueWorkerStop()
	close(errorsChan)
	queueWorkerErr := bytes.Buffer{}
	for c := range errorsChan {
		if queueWorkerErr.String() == "" {
			queueWorkerErr.WriteString("worker err=>")
		}
		queueWorkerErr.WriteString(c.Error())
	}
	if queueWorkerErr.String() != "" {
		err = fmt.Errorf(queueWorkerErr.String())
	}

	return err
}

// setupQueueVars ...
func setupQueueVars(queueApp *kelvins.QueueApplication) error {
	var logger log.LoggerContextIface
	if kelvins.ServerSetting != nil {
		switch kelvins.ServerSetting.Environment {
		case config.DefaultEnvironmentDev:
			logger = kelvins.AccessLogger
		case config.DefaultEnvironmentTest:
			logger = kelvins.AccessLogger
		default:
		}
	}
	if logger != nil {
		queueLog.Set(&queueLogger{
			logger: logger,
		})
	}

	// only queueApp need check GetNamedTaskFuncs or RegisterEventHandler
	if queueApp.GetNamedTaskFuncs == nil && queueApp.RegisterEventHandler == nil {
		return fmt.Errorf("lack of implement GetNamedTaskFuncs And RegisterEventHandler")
	}
	err := setupCommonQueue(queueApp.GetNamedTaskFuncs())
	if err != nil {
		return err
	}

	return nil
}

func queueWorkerStop() {
	//for queue,worker := range queueWorker {
	//	// process exit queue worker should exit
	//	// worker.Quit()
	//	// return
	//}
	logging.Info("queueApp queue worker stop over")
}

var queueLoggerCtx = context.Background()

// queueLogger implements machinery log interface.
type queueLogger struct {
	logger log.LoggerContextIface
}

// Print uses logger to log info msg.
func (q *queueLogger) Print(a ...interface{}) {
	q.logger.Info(queueLoggerCtx, fmt.Sprint(a...))
}

// Printf uses logger to log info msg.
func (q *queueLogger) Printf(format string, a ...interface{}) {
	q.logger.Infof(queueLoggerCtx, format, a...)
}

// Println uses logger to log info msg.
func (q *queueLogger) Println(a ...interface{}) {
	q.logger.Info(queueLoggerCtx, fmt.Sprint(a...))
}

// Fatal uses kelvins.ErrLogger to log err msg.
func (q *queueLogger) Fatal(a ...interface{}) {
	q.logger.Error(queueLoggerCtx, fmt.Sprint(a...))
}

// Fatalf uses kelvins.ErrLogger to log err msg.
func (q *queueLogger) Fatalf(format string, a ...interface{}) {
	q.logger.Errorf(queueLoggerCtx, format, a...)
}

// Fatalln uses kelvins.ErrLogger to log err msg.
func (q *queueLogger) Fatalln(a ...interface{}) {
	q.logger.Error(queueLoggerCtx, fmt.Sprint(a...))
}

// Panic uses kelvins.ErrLogger to log err msg.
func (q *queueLogger) Panic(a ...interface{}) {
	q.logger.Error(queueLoggerCtx, fmt.Sprint(a...))
}

// Panicf uses kelvins.ErrLogger to log err msg.
func (q *queueLogger) Panicf(format string, a ...interface{}) {
	q.logger.Errorf(queueLoggerCtx, format, a)
}

// Panicln uses kelvins.ErrLogger to log err msg.
func (q *queueLogger) Panicln(a ...interface{}) {
	q.logger.Error(queueLoggerCtx, fmt.Sprint(a...))
}
