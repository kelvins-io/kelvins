package app

import (
	"bytes"
	"context"
	"fmt"
	"gitee.com/kelvins-io/common/convert"
	"gitee.com/kelvins-io/common/event"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/setup"
	"gitee.com/kelvins-io/kelvins/util/kprocess"
	"github.com/RichardKnop/machinery/v1"
	queueLog "github.com/RichardKnop/machinery/v1/log"
	"time"
)

// RunQueueApplication runs queue application.
func RunQueueApplication(application *kelvins.QueueApplication) {
	if application.Name == "" {
		logging.Fatal("Application name can't not be empty")
	}
	application.Type = kelvins.AppTypeQueue

	showAppVersion(application.Application)

	err := runQueue(application)
	if err != nil {
		logging.Infof("RunQueueApplication err: %v\n", err)
	}

	appPrepareForceExit()
	// Wait for connections to drain.
	err = appShutdown(application.Application)
	if err != nil {
		logging.Infof("App.appShutdown err: %v\n", err)
	}
	logging.Info("App appShutdown over")
}

var queueWorker =map[*machinery.Worker]struct{}{}

// runQueue runs queue application.
func runQueue(queueApp *kelvins.QueueApplication) error {

	// 1. load config
	err := config.LoadDefaultConfig(queueApp.Application)
	if err != nil {
		return err
	}
	if queueApp.LoadConfig != nil {
		err = queueApp.LoadConfig()
		if err != nil {
			return err
		}
	}

	// 2. init application
	err = initApplication(queueApp.Application)
	if err != nil {
		return err
	}

	// 3. setup vars
	err = setupCommonVars(queueApp.Application)
	if err != nil {
		return err
	}
	err = setupQueueVars(queueApp)
	if err != nil {
		return err
	}
	if queueApp.SetupVars != nil {
		err = queueApp.SetupVars()
		if err != nil {
			return err
		}
	}

	// 4  startup control
	next, err := startUpControl(kelvins.PIDFile)
	if err != nil {
		return err
	}
	if !next {
		return nil
	}

	// 5. apollo hot update listen
	//config.TriggerApolloHotUpdateListen(queueApp.Application)

	// 6. event server
	if queueApp.EventServer != nil {
		logging.Info("Start event server consume")
		// subscribe event
		if queueApp.RegisterEventHandler != nil {
			err := queueApp.RegisterEventHandler(queueApp.EventServer)
			if err != nil {
				return err
			}
		}
		// start event server
		err = queueApp.EventServer.Start()
		if err != nil {
			return err
		}
		logging.Info("Start event server")
	}

	// 7. queue server
	logging.Info("Start queue server consume")
	concurrency := len(queueApp.GetNamedTaskFuncs())
	if kelvins.QueueServerSetting != nil {
		concurrency = kelvins.QueueServerSetting.WorkerConcurrency
	}
	logging.Infof("Count of worker goroutine: %d\n", concurrency)
	consumerTag := queueApp.Application.Name + convert.Int64ToStr(time.Now().Local().UnixNano())

	kp := new(kprocess.KProcess)
	_, err = kp.Listen("", "", kelvins.PIDFile)
	if err != nil {
		return fmt.Errorf("KProcess listen err: %v", err)
	}
	var queueList = []string{""}
	queueList = append(queueList, kelvins.QueueServerSetting.CustomQueueList...)

	errorsChan := make(chan error,len(queueList))
	for _, customQueue := range queueList {
		cTag := consumerTag
		if len(customQueue) > 0 {
			cTag = customQueue + "-" + consumerTag
		}
		logging.Infof("Consumer Tag: %s\n", cTag)
		worker := queueApp.QueueServer.TaskServer.NewCustomQueueWorker(cTag, concurrency, customQueue)
		worker.LaunchAsync(errorsChan)
		queueWorker[worker] = struct{}{}
	}

	<-kp.Exit() // worker not listen Interrupt,SIGTERM signal stop

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
	var err error
	queueApp.QueueLogger, err = log.GetBusinessLogger("queue.consume")
	if err != nil {
		return fmt.Errorf("kelvinslog.GetBusinessLogger: %v", err)
	}
	queueLog.Set(&queueLogger{
		logger: queueApp.QueueLogger,
	})

	if queueApp.GetNamedTaskFuncs == nil && queueApp.RegisterEventHandler == nil {
		return fmt.Errorf("lack of implement GetNamedTaskFuncs And RegisterEventHandler")
	}
	if kelvins.QueueRedisSetting != nil && kelvins.QueueRedisSetting.Broker != "" {
		queueApp.QueueServer, err = setup.NewRedisQueue(kelvins.QueueRedisSetting, queueApp.GetNamedTaskFuncs())
		if err != nil {
			return err
		}
		return nil
	}
	if kelvins.QueueAMQPSetting != nil && kelvins.QueueAMQPSetting.Broker != "" {
		queueApp.QueueServer, err = setup.NewAMQPQueue(kelvins.QueueAMQPSetting, queueApp.GetNamedTaskFuncs())
		if err != nil {
			return err
		}
		return nil
	}
	if kelvins.QueueAliAMQPSetting != nil && kelvins.QueueAliAMQPSetting.VHost != "" {
		queueApp.QueueServer, err = setup.NewAliAMQPQueue(kelvins.QueueAliAMQPSetting, queueApp.GetNamedTaskFuncs())
		if err != nil {
			return err
		}
		return nil
	}
	// init event server
	if kelvins.AliRocketMQSetting != nil && kelvins.AliRocketMQSetting.InstanceId != "" {
		logger, err := log.GetBusinessLogger("event")
		if err != nil {
			return err
		}

		// new event server
		eventServer, err := event.NewEventServer(&event.Config{
			BusinessName: kelvins.AliRocketMQSetting.BusinessName,
			RegionId:     kelvins.AliRocketMQSetting.RegionId,
			AccessKey:    kelvins.AliRocketMQSetting.AccessKey,
			SecretKey:    kelvins.AliRocketMQSetting.SecretKey,
			InstanceId:   kelvins.AliRocketMQSetting.InstanceId,
			HttpEndpoint: kelvins.AliRocketMQSetting.HttpEndpoint,
		}, logger)
		if err != nil {
			return err
		}

		queueApp.EventServer = eventServer
		return nil
	}

	return fmt.Errorf("lack of kelvinsQueue* section config")
}

func queueWorkerStop()  {
	for q := range queueWorker {
		if q != nil {
			// process exit worker should exit
			//q.Quit()
			//return
		}
	}
	logging.Info("queue worker stop over")
}

var queueLoggerCtx = context.Background()

// queueLogger implements machinery log interface.
type queueLogger struct {
	logger *log.LoggerContext
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
