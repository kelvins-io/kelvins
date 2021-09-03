package app

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/common/event"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/util/kprocess"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"time"
)

// RunCronApplication runs cron application.
func RunCronApplication(application *kelvins.CronApplication) {
	application.Type = kelvins.AppTypeCron

	err := prepareCron(application)
	if err != nil {
		logging.Infof("CronApp prepareCron err: %v\n", err)
	}

	appPrepareForceExit()
	if application.Cron != nil {
		application.Cron.Stop()
		logging.Info("CronApp Task Stop over")
	}
	err = appShutdown(application.Application)
	if err != nil {
		logging.Infof("CronApp appShutdown err: %v\n", err)
	}
	logging.Info("CronApp appShutdown over")
}

// prepareCron prepares cron application.
func prepareCron(cronApp *kelvins.CronApplication) error {
	var err error

	// 1. init application
	err = initApplication(cronApp.Application)
	if err != nil {
		return err
	}

	// 2 init cron vars
	err = setupCronVars(cronApp)
	if err != nil {
		return err
	}

	// 3  register event handler
	if cronApp.EventServer != nil && cronApp.RegisterEventHandler != nil {
		logging.Info("Start event server consume")
		// subscribe event
		if cronApp.RegisterEventHandler != nil {
			err := cronApp.RegisterEventHandler(cronApp.EventServer)
			if err != nil {
				return err
			}
		}
		// start event server
		err = cronApp.EventServer.Start()
		if err != nil {
			return err
		}
		logging.Info("Start event server")
	}

	// 4. register cron jobs
	if cronApp.GenCronJobs != nil {
		cronJobs := cronApp.GenCronJobs()
		if len(cronJobs) != 0 {
			jobNameDict := map[string]int{}
			for _, j := range cronJobs {
				if j.Name == "" {
					return fmt.Errorf("Lack of CronJob.Name")
				}
				if j.Spec == "" {
					return fmt.Errorf("Lack of CronJob.Spec")
				}
				if j.Job == nil {
					return fmt.Errorf("Lack of CronJob.Job")
				}
				if _, ok := jobNameDict[j.Name]; ok {
					return fmt.Errorf("Repeat job name: %s", j.Name)
				}
				jobNameDict[j.Name] = 1
				job := &cronJob{
					logger: cronApp.CronLogger,
					name:   j.Name,
				}
				_, err = cronApp.Cron.AddFunc(j.Spec, job.warpJob(j.Job))
				if err != nil {
					return fmt.Errorf("Cron.AddFunc err: %v", err)
				}
			}
		}
	}

	// 5. run cron app
	kp := new(kprocess.KProcess)
	_, err = kp.Listen("", "", kelvins.PIDFile)
	if err != nil {
		return fmt.Errorf("KProcess listen err: %v", err)
	}
	logging.Info("Start cron task")
	cronApp.Cron.Start()

	<-kp.Exit()

	return nil
}

// setupCronVars ...
func setupCronVars(cronApp *kelvins.CronApplication) error {
	var err error
	cronApp.CronLogger, err = log.GetBusinessLogger("cron.schedule")
	if err != nil {
		return err
	}
	cronApp.Cron = cron.New(cron.WithSeconds())

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

		cronApp.EventServer = eventServer
		return nil
	}

	return nil
}

// cronJob ...
type cronJob struct {
	name   string
	logger *log.LoggerContext
}

var cronJobCtx = context.Background()

// warpJob warps job with log and panic recover.
func (c *cronJob) warpJob(job func()) func() {
	return func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Errorf(cronJobCtx, "Name: %s Recover err: %v", c.name, r)
			}
		}()
		UUID := uuid.New()
		startTime := time.Now()
		c.logger.Infof(cronJobCtx, "Name: %s Uuid: %s StartTime: %s", c.name, UUID, startTime.Format("2006-01-02 15:04:05.000"))
		job()
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		c.logger.Infof(cronJobCtx, "Name: %s Uuid: %s EndTime: %s Duration: %fs", c.name, UUID, endTime.Format("2006-01-02 15:04:05.000"), duration.Seconds())
	}
}
