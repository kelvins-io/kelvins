package app

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/common/event"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/util/kprocess"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"time"
)

// RunCronApplication runs cron application.
func RunCronApplication(application *kelvins.CronApplication) {
	if application.Name == "" {
		logging.Fatal("Application name can't not be empty")
	}
	application.Type = kelvins.AppTypeCron

	err := prepareCron(application)
	if err != nil {
		logging.Fatalf("prepareCron err: %v", err)
	}

	// kprocess listen
	_, err = kprocess.Listen("", "", kelvins.PIDFile)
	if err != nil {
		logging.Fatalf("kprocess listen err: %v", err)
	}
	logging.Info("Start cron")
	application.Cron.Start()
	<-kprocess.Exit()

	appPrepareForceExit()
	application.Cron.Stop()
	err = appShutdown(application.Application)
	if err != nil {
		logging.Fatalf("App.appShutdown err: %v", err)
	}
}

// prepareCron prepares cron application.
func prepareCron(cronApp *kelvins.CronApplication) error {

	// 1. load config
	err := config.LoadDefaultConfig(cronApp.Application)
	if err != nil {
		return err
	}
	if cronApp.LoadConfig != nil {
		err = cronApp.LoadConfig()
		if err != nil {
			return err
		}
	}

	// 2. init application
	err = initApplication(cronApp.Application)
	if err != nil {
		return err
	}

	// 3. setup vars
	err = setupCronVars(cronApp)
	if err != nil {
		return err
	}
	if cronApp.SetupVars != nil {
		err = cronApp.SetupVars()
		if err != nil {
			return err
		}
	}

	// 4  register event handler
	if cronApp.EventServer != nil && cronApp.RegisterEventHandler != nil {
		logging.Infof("Start event server consume")
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

	// 5. register cron jobs
	if cronApp.GenCronJobs != nil {
		cronJobs := cronApp.GenCronJobs()
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

	return nil
}

// setupCronVars ...
func setupCronVars(cronApp *kelvins.CronApplication) error {
	err := setupCommonVars(cronApp.Application)
	if err != nil {
		return err
	}

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

// warpJob warps job with log and panic recover.
func (c *cronJob) warpJob(job func()) func() {
	return func() {
		defer func() {
			if r := recover(); r != nil {
				kelvins.ErrLogger.Errorf(context.Background(), "Name: %s Recover err: %v", c.name, r)
			}
		}()
		UUID := uuid.New()
		startTime := time.Now()
		c.logger.Infof(context.Background(), "Name: %s Uuid: %s StartTime: %s", c.name, UUID, startTime.Format("2006-01-02 15:04:05.000"))
		job()
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		c.logger.Infof(context.Background(), "Name: %s Uuid: %s EndTime: %s Duration: %fs", c.name, UUID, endTime.Format("2006-01-02 15:04:05.000"), duration.Seconds())
	}
}
