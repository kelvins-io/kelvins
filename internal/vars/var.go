package vars

import "gitee.com/kelvins-io/common/log"

// Version is internal vars fork root path vars.go
var Version = "1.5.x"

// FrameworkLogger is a internal var for Framework log
var FrameworkLogger log.LoggerContextIface

// ErrLogger is a internal vars for application to log err msg.
var ErrLogger log.LoggerContextIface

// AccessLogger is a internal vars for application to log access log
var AccessLogger log.LoggerContextIface

// BusinessLogger is a internal vars for application to log business log
var BusinessLogger log.LoggerContextIface

// AppCloseCh is a internal vars for app close notice
var AppCloseCh chan struct{}

// ServiceIp is current service ip addr
var ServiceIp string

// ServicePort is current service port
var ServicePort string
