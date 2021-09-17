package app

import (
	"context"
	"fmt"
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
	if application == nil || application.Application == nil {
		panic("cronApplication is nil or application is nil")
	}
	// app instance once validate
	{
		err := appInstanceOnceValidate()
		if err != nil {
			logging.Fatal(err.Error())
		}
	}

	application.Type = kelvins.AppTypeCron
	kelvins.CronAppInstance = application

	err := runCron(application)
	if err != nil {
		logging.Infof("cronApp runCron err: %v\n", err)
	}

	appPrepareForceExit()
	if application.Cron != nil {
		application.Cron.Stop()
		logging.Info("cronApp Task Stop over")
	}
	err = appShutdown(application.Application, 0)
	if err != nil {
		logging.Infof("cronApp appShutdown err: %v\n", err)
	}
	logging.Info("cronApp appShutdown over")
}

// runCron prepares cron application.
func runCron(cronApp *kelvins.CronApplication) error {
	var err error

	// 1. init application
	err = initApplication(cronApp.Application)
	if err != nil {
		return err
	}
	if !appProcessNext {
		return err
	}

	// 2 init cron vars
	err = setupCronVars(cronApp)
	if err != nil {
		return err
	}

	// 3  register event handler
	if kelvins.EventServerAliRocketMQ != nil {
		logging.Info("cronApp Start event server")
		if cronApp.RegisterEventProducer != nil {
			appRegisterEventProducer(cronApp.RegisterEventProducer, cronApp.Type)
		}
		if cronApp.RegisterEventHandler != nil {
			appRegisterEventHandler(cronApp.RegisterEventHandler, cronApp.Type)
		}
	}

	// 4. register cron jobs
	if cronApp.GenCronJobs != nil {
		cronJobs := cronApp.GenCronJobs()
		if len(cronJobs) != 0 {
			jobNameDict := map[string]int{}
			for _, j := range cronJobs {
				if j.Name == "" {
					return fmt.Errorf("lack of CronJob.Name")
				}
				if j.Spec == "" {
					return fmt.Errorf("lack of CronJob.Spec")
				}
				if j.Job == nil {
					return fmt.Errorf("lack of CronJob.Job")
				}
				if _, ok := jobNameDict[j.Name]; ok {
					return fmt.Errorf("repeat job name: %s", j.Name)
				}
				jobNameDict[j.Name] = 1
				job := &cronJob{
					name: j.Name,
				}
				var logger log.LoggerContextIface
				if kelvins.ServerSetting != nil {
					switch kelvins.ServerSetting.Environment {
					case config.DefaultEnvironmentDev:
						logger = kelvins.BusinessLogger
					case config.DefaultEnvironmentTest:
						logger = kelvins.BusinessLogger
					default:
					}
				}
				job.logger = logger
				_, err = cronApp.Cron.AddFunc(j.Spec, job.warpJob(j.Job))
				if err != nil {
					return fmt.Errorf("addFunc err: %v", err)
				}
			}
		}
	}

	// 5. run cron app
	kp := new(kprocess.KProcess)
	_, err = kp.Listen("", "", kelvins.PIDFile)
	if err != nil {
		return fmt.Errorf("kprocess listen pidFile(%v) err: %v", kelvins.PIDFile, err)
	}
	logging.Info("cronApp Start cron task")
	cronApp.Cron.Start()

	<-kp.Exit()

	return nil
}

// setupCronVars ...
func setupCronVars(cronApp *kelvins.CronApplication) error {
	cronApp.Cron = cron.New(cron.WithSeconds())

	err := setupCommonQueue(nil)
	if err != nil {
		return err
	}

	return nil
}

// cronJob ...
type cronJob struct {
	name   string
	logger log.LoggerContextIface
}

var cronJobCtx = context.Background()

// warpJob warps job with log and panic recover.
func (c *cronJob) warpJob(job func()) func() {
	return func() {
		defer func() {
			if r := recover(); r != nil {
				if c.logger != nil {
					c.logger.Errorf(cronJobCtx, "cron Job name: %s recover err: %v", c.name, r)
				} else {
					logging.Infof("cron Job name: %s recover err: %v\n", c.name, r)
				}
			}
		}()
		UUID := uuid.New()
		startTime := time.Now()
		if c.logger != nil {
			c.logger.Infof(cronJobCtx, "Name: %s Uuid: %s StartTime: %s",
				c.name, UUID, startTime.Format("2006-01-02 15:04:05.000"))
		}
		job()
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		if c.logger != nil {
			c.logger.Infof(cronJobCtx, "Name: %s Uuid: %s EndTime: %s Duration: %fs",
				c.name, UUID, endTime.Format("2006-01-02 15:04:05.000"), duration.Seconds())
		}
	}
}
