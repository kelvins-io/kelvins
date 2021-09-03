package vars

import "gitee.com/kelvins-io/common/log"

// Version this vars fork root path vars.go
var Version = "1.5.0"

// FrameworkLogger is a global var for Framework log
var FrameworkLogger log.LoggerContextIface

// ErrLogger is a global vars for application to log err msg.
var ErrLogger log.LoggerContextIface

// AccessLogger is a global vars for application to log access log
var AccessLogger log.LoggerContextIface

// BusinessLogger is a global vars for application to log business log
var BusinessLogger log.LoggerContextIface
